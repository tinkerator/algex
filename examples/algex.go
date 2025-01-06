// Program algex is an interactive text based program to explore
// algebra. It is a command line interface to a suite of Go packages
// to perform algebraic manipulation.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"zappem.net/pub/io/lined"
	"zappem.net/pub/math/algex/factor"
	"zappem.net/pub/math/algex/terms"
)

var (
	tok    = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_]*|[0-9]+|\:\=|[-+*/^=(),%]|\s*|#.*)`)
	space  = regexp.MustCompile(`^\s+$`)
	symbol = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

	filer = flag.String("file", "", "name of algex (.ax) script to start with")
)

// split tokenizes the input.
func split(line string) (toks []string) {
	for i := 0; i < len(line); i++ {
		loc := tok.FindStringIndex(line[i:])
		if loc == nil || loc[1] == 0 {
			toks = append(toks, line[i:])
			break
		}
		start := i + loc[0]
		end := i + loc[1]
		i = end - 1
		if space.MatchString(line[start:end]) {
			continue
		}
		toks = append(toks, line[start:end])
	}
	return toks
}

// exp returns a single expression, fully expanded.
func exp(toks []string) (*terms.Frac, []*terms.Frac, error) {
	return terms.ParseFrac(strings.Join(toks, " "))
}

// build returns a fractional expression which is completely expanded.
// Anticipating groups of expressions, the build return an array of
// such fractions. However, at present, it can't recognize such an
// array.
func build(toks []string) (es []*terms.Frac, err error) {
	e, as, err2 := exp(toks)
	if err2 != nil {
		err = err2
		return
	}
	if as != nil {
		es = as
		return
	}
	es = []*terms.Frac{e}
	return
}

func helpInfo() {
	fmt.Println("Algex (c) 2023,24,25 tinkerer@zappem.net\n")
	fmt.Println(`Commands:

<exp>		display expression (simplified)
file <name>	take commands from a named file
xxx := <exp>	learn a simple substitution for simplification
xxx = <exp>	learn a simplified substitution for simplification
list		list all of the known substitutions
reduce <exp>    express as simple expression plus a remainder
exit		exit the program
help		this message
<exp> mod <n>   compute modular result for expressions with a denominator of 1`)
}

// Vars is used to make substitutions.
type Vars struct {
	fact  []factor.Value
	subst *terms.Frac
}

// inline substitutes values a number of times to simplify an
// expression with the known replacement values.
//
// TODO pick a better strategy for simplification. Perhaps order
// substitutions by smallness of substitution sets?
func inline(f *terms.Frac, vars map[string]*Vars) *terms.Frac {
	var vs []string
	for v := range vars {
		vs = append(vs, v)
	}
	sort.Slice(vs, func(a, b int) bool {
		as, bs := vars[vs[a]], vars[vs[b]]
		am, bm := factor.Order(as.fact), factor.Order(bs.fact)
		if am == bm {
			return vs[a] < vs[b]
		}
		return am > bm
	})
	for i := 0; i < 8; i++ {
		changed := false
		for _, v := range vs {
			vv := vars[v]
			var modified bool
			f, modified = f.Substituted(vv.fact, vv.subst)
			changed = changed || modified
		}
		if !changed {
			break
		}
	}
	return f
}

func main() {
	flag.Parse()

	vars := make(map[string]*Vars)

	t := lined.NewReader()
	var f *os.File
	var fs []*os.File
	var reading *bufio.Scanner
	var files []*bufio.Scanner

	if *filer != "" {
		f, err := os.Open(*filer)
		if err != nil {
			fmt.Printf("unable to open %q: %v\n", *filer, err)
			os.Exit(1)
		}
		fs = append(fs, f)
		reading = bufio.NewScanner(f)
		files = append(files, reading)
	}

	for {
		var line string
		var err error

		if reading != nil {
			if !reading.Scan() {
				f.Close()
				fs = fs[:len(fs)-1]
				files = files[:len(files)-1]
				if len(files) == 0 {
					reading = nil
					f = nil
				} else {
					reading = files[len(files)-1]
					f = fs[len(fs)-1]
				}
				continue
			}
			line = reading.Text()
		} else {
			fmt.Print("> ")
			line, err = t.ReadString()
			if err != nil {
				switch err {
				default:
					log.Fatalf("unable to recover: %v", err)
				}
				continue
			}
		}

		toks := split(line)
		if len(toks) == 0 {
			continue
		}

		if len(toks) == 1 {
			switch toks[0] {
			case "exit":
				fmt.Println("exiting")
				os.Exit(0)
			case "list":
				var ts []string
				for k := range vars {
					ts = append(ts, k)
				}
				sort.Strings(ts)
				for _, k := range ts {
					fmt.Printf(" %s := %v\n", k, vars[k].subst)
				}
				continue
			case "help":
				helpInfo()
				continue
			default:
				if strings.HasPrefix(toks[0], "#") {
					// ignore comment
					continue
				}
			}
			// fall through - this is an expression.
		} else if toks[0] == "file" {
			path := strings.TrimSpace(line[5 : len(line)-1])
			f, err = os.Open(path)
			if err != nil {
				fmt.Printf("unable to open %q: %v\n", path, err)
				continue
			}
			reading = bufio.NewScanner(f)
			fs = append(fs, f)
			files = append(files, reading)
			continue
		} else if toks[0] == "reduce" {
			es, err := build(toks[1:])
			if err != nil {
				fmt.Printf("expression problem: %v\n", err)
				continue
			}
			for _, e := range es {
				e = inline(e, vars)
				a, b, err := e.Num.Divide(e.Den)
				if err != nil {
					fmt.Printf(" %v\n", e)
					continue
				}
				fmt.Printf(" %v rem %v\n", a, terms.NewFrac(b, e.Den))
			}
			continue
		} else {
			var op string
			var before, after []string
			// scan for ":=", "=" or "mod"
			for i := 1; op == "" && i < len(toks); i++ {
				switch toks[i] {
				case ":=", "=", "mod":
					op = toks[i]
					before = toks[:i]
					after = toks[i+1:]
				}
			}
			if op != "" {
				lhs, err := build(before)
				if err != nil || len(lhs) != 1 {
					fmt.Printf("invalid pre-op (%q) expression, %q: %v\n", op, before, err)
					continue
				}
				if len(after) == 0 {
					te, err := lhs[0].Num.Leading()
					if err != nil {
						fmt.Printf("invalid expression to drop from list: %q\n", lhs)
						continue
					}
					delete(vars, factor.Prod(te.Fact...))
					continue
				}
				rhs, err := build(after)
				if err != nil || len(rhs) != 1 {
					fmt.Printf("invalid post-op (%q) expression: %q\n", op, after)
					continue
				}
				left := lhs[0].Num
				right := terms.NewFrac(rhs[0].Num.Mul(lhs[0].Den), rhs[0].Den)
				switch op {
				case ":=", "=":
					te, err := left.Leading()
					if err != nil {
						fmt.Printf("pre-op (%q) expression must have one non-numeric term: %q does not\n", op, left)
						continue
					}
					fe := te.Exp()
					ce := terms.Rat(te.Coeff)
					nr := terms.NewFrac(right.Num, fe.Sub(left), ce.Mul(right.Den))
					if op == "=" {
						nr = inline(nr, vars)
					} else {
						nr.Reduce()
					}
					vars[factor.Prod(te.Fact...)] = &Vars{
						fact:  te.Fact,
						subst: nr,
					}
					continue
				case "mod":
					if lhs[0].Den.String() != "1" {
						fmt.Printf("expression, %v, has denominator - no mod value\n", lhs)
						continue
					}
					if rhs[0].Den.String() != "1" {
						fmt.Printf("modulus, %v, has denominator - no mod value\n", rhs)
						continue
					}
					m, ok := rhs[0].Num.AsNumber()
					if !ok {
						fmt.Printf("modulus, %v, is not a number", rhs)
					}
					f := inline(lhs[0], vars)
					fmt.Printf(" %v\n", f.Num.Mod(factor.R(m)))
					continue
				}
			}
		}

		es, err := build(toks)
		if err != nil {
			fmt.Printf("syntax error %q: %v\n", toks, err)
			continue
		}
		for _, e := range es {
			fmt.Printf(" %v\n", inline(e, vars))
		}
	}
}
