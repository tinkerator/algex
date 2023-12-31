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
func exp(toks []string) (*terms.Frac, error) {
	return terms.ParseFrac(strings.Join(toks, " "))
}

// build returns a list of comma separated expressions. Each expression
// is completely expanded.
func build(toks []string) (es []*terms.Frac, err error) {
	j := 0
outer:
	for ; j < len(toks); j++ {
		for i, t := range toks[j:] {
			if t == "," {
				e, err2 := exp(toks[j : j+i])
				if err2 != nil {
					es = nil
					err = err2
					return
				}
				es = append(es, e)
				j += i
				continue outer
			}
		}
		e, err2 := exp(toks[j:])
		if err2 != nil {
			es = nil
			err = err2
			return
		}
		es = append(es, e)
		break
	}
	return
}

func helpInfo() {
	fmt.Println("Algex (c) 2023 tinkerer@zappem.net\n")
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

// inline substitutes values a number of times to simplify an
// expression with the known replacement values.
//
// TODO pick a better strategy for simplification. Perhaps order
// substitutions by smallness of substitution sets?
func inline(f *terms.Frac, vars map[string]*terms.Frac) *terms.Frac {
	for i := 0; i < 4; i++ {
		for s, v := range vars {
			f = f.Substitute([]factor.Value{factor.S(s)}, v)
		}
	}
	return f
}

func main() {
	flag.Parse()

	vars := make(map[string]*terms.Frac)

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
					fmt.Printf(" %s := %v\n", k, vars[k])
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
			switch toks[1] {
			case ":=", "=":
				if !symbol.MatchString(toks[0]) {
					fmt.Printf("invalid assignment to %q\n", toks[0])
					continue
				}
				if len(toks) < 3 {
					delete(vars, toks[0])
					continue
				}
				es, err := build(toks[2:])
				if err != nil {
					fmt.Printf("assignment failed: %v\n", err)
					continue
				}
				if len(es) != 1 {
					fmt.Printf("unable to assign multiple expressions to %q\n", toks[0])
					continue
				}
				if toks[1] == "=" {
					es[0] = inline(es[0], vars)
				}
				vars[toks[0]] = es[0]
				continue
			case "mod":
				s := toks[0]
				f, ok := vars[s]
				if !ok {
					fmt.Printf("invalid modular to %q\n", s)
					continue
				}
				if len(toks) != 3 {
					fmt.Println("usage: <exp> mod <number>")
					continue
				}
				m, k, err := factor.Parse(toks[2])
				if err != nil || k != len(toks[2]) {
					fmt.Printf("invalid modular denominator: %q\n", toks[2:])
					continue
				}
				f = inline(f, vars)
				if f.Den.String() != "1" {
					fmt.Printf("expression, %v, has denominator - no mod value\n", f)
					continue
				}
				fmt.Printf(" %v\n", f.Num.Mod(m[0]))
				continue
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
