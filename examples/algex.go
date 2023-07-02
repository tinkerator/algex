// Program algex is an interactive text based program to explore
// algebra. It is a command line interface to a suite of Go packages
// to perform algebraic manipulation.
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"zappem.net/math/algex/factor"
	"zappem.net/math/algex/terms"
	"zappem.net/pub/io/lined"
)

var (
	tok    = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_]*|[0-9]+|\:\=|[-+*/^=(),%]|\s*|#.*)`)
	space  = regexp.MustCompile(`^\s+$`)
	symbol = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
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
func exp(toks []string) (*terms.Exp, error) {
	// TODO handle "(" and ")" parsing.
	return terms.ParseExp(strings.Join(toks, " "))
}

// build returns a list of comma separated expressions. Each expression
// is completely expanded.
func build(toks []string) (es []*terms.Exp, err error) {
	j := 0
outer:
	for ; j < len(toks); j++ {
		for i, t := range toks[j:] {
			if t == "," {
				e, err2 := exp(toks[j:i])
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

func main() {
	fmt.Printf("Algex (c) 2023 tinkerer@zappem.net\n\n")
	vars := make(map[string]*terms.Exp)

	t := lined.NewReader()
	for {
		fmt.Print("> ")
		line, err := t.ReadString()
		if err != nil {
			switch err {
			default:
				log.Fatalf("unable to recover: %v", err)
			}
			continue
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
			default:
				if strings.HasPrefix(toks[0], "#") {
					// ignore comment
					continue
				}
			}
		} else {
			switch toks[1] {
			case ":=":
				if !symbol.MatchString(toks[0]) {
					fmt.Printf("invalid assignment to %q\n", toks[0])
					continue
				}
				var es []*terms.Exp
				if len(toks) > 4 && toks[2] == "subst" && toks[4] == "," {
					if e, ok := vars[toks[3]]; !ok {
						fmt.Printf("variable %q not defined\n", toks[3])
						continue
					} else {
						es = []*terms.Exp{
							vars[toks[5]].Substitute([]factor.Value{factor.S(toks[3])}, e),
						}
					}
				} else {
					// (un)define a value.
					// TODO verify that es does not contain toks[0].
					es, err = build(toks[2:])
					if err != nil {
						fmt.Printf("assignment to %q failed: %v\n", toks[0], err)
						continue
					}
				}
				switch len(es) {
				case 0:
					delete(vars, toks[0])
				case 1:
					vars[toks[0]] = es[0]
				default:
					fmt.Printf("assignment limited to single RHS value: %v\n", es)
				}
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
				for i := 0; i < 3; i++ {
					for s, v := range vars {
						f = f.Substitute([]factor.Value{factor.S(s)}, v)
					}
				}
				fmt.Printf(" %v\n", f.Mod(m[0]))
				continue
			}
		}

		es, err := build(toks)
		if err != nil {
			fmt.Printf("syntax error %q: %v\n", toks, err)
			continue
		}
		// TODO pick a better strategy for simplification.
		// Order substitutions by smallness of substitution sets?
		for _, e := range es {
			f := e
			for i := 0; i < 3; i++ {
				for s, v := range vars {
					f = f.Substitute([]factor.Value{factor.S(s)}, v)
				}
			}
			fmt.Printf(" %v\n", f)
		}
	}
}
