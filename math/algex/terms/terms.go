// Package terms abstracts sums of products of factors.
package terms

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"zappem.net/pub/math/algex/factor"
)

// Term is a product of a coefficient and a set of non-numerical factors.
type Term struct {
	Coeff *big.Rat
	Fact  []factor.Value
}

// Exp is a an expression or sum of terms.
type Exp struct {
	terms map[string]Term
}

// NewExp creates a new expression.
func NewExp(ts ...[]factor.Value) *Exp {
	e := &Exp{
		terms: make(map[string]Term),
	}
	for _, t := range ts {
		n, fs, s := factor.Segment(t...)
		if n == nil {
			continue
		}
		e.insert(n, fs, s)
	}
	return e
}

// IsZero confirms a simplified expression is zero.
func (e *Exp) IsZero() bool {
	return e == nil || len(e.terms) == 0
}

// String represents an expression of Terms as a string.
func (e *Exp) String() string {
	if e.IsZero() {
		return "0"
	}
	var s []string
	for x := range e.terms {
		s = append(s, x)
	}
	// TODO: might want to prefer a non-ascii sorted expression.
	sort.Strings(s)
	for i, x := range s {
		f := e.terms[x]
		v := []factor.Value{factor.R(f.Coeff)}
		t := factor.Prod(append(v, f.Fact...)...)
		if i != 0 && t[0] != '-' {
			s[i] = "+" + t
		} else {
			s[i] = t
		}
	}
	return strings.Join(s, "")
}

// Int generates an expression of a constant integer.
func Int(n *big.Int) *Exp {
	return NewExp([]factor.Value{factor.I(n)})
}

// insert merges a coefficient, a product of factors to an expression
// indexed by s.
func (e *Exp) insert(n *big.Rat, fs []factor.Value, s string) {
	old, ok := e.terms[s]
	if !ok {
		e.terms[s] = Term{
			Coeff: n,
			Fact:  fs,
		}
		return
	}
	// Combine with existing term.
	old.Coeff = n.Add(n, e.terms[s].Coeff)
	if old.Coeff.Cmp(&big.Rat{}) == 0 {
		delete(e.terms, s)
		return
	}
	e.terms[s] = old
}

// Sum adds together expressions. With only one argument, Add is a
// simple duplicate function.
func Sum(as ...*Exp) *Exp {
	e := &Exp{
		terms: make(map[string]Term),
	}
	for _, a := range as {
		if a == nil {
			continue
		}
		for s, t := range a.terms {
			m := &big.Rat{}
			e.insert(m.Set(t.Coeff), t.Fact, s)
		}
	}
	return e
}

// Add adds together two expressions and returns a single expression:
// a+b.
func (a *Exp) Add(b *Exp) *Exp {
	return Sum(a, b)
}

// Sub subtracts b from a into a new expression.
func (a *Exp) Sub(b *Exp) *Exp {
	e := &Exp{
		terms: make(map[string]Term),
	}
	if a != nil {
		for s, t := range a.terms {
			m := &big.Rat{}
			e.insert(m.Set(t.Coeff), t.Fact, s)
		}
	}
	for s, t := range b.terms {
		m := big.NewRat(-1, 1)
		e.insert(m.Mul(m, t.Coeff), t.Fact, s)
	}
	return e
}

var zero = []factor.Value{factor.R(&big.Rat{})}

// Mod takes a numerical integer factor and eliminates obvious
// multiples of it from an expression. No attempt is made to
// simplify non-integer fractions.
func (e *Exp) Mod(x factor.Value) *Exp {
	if !x.IsNum() || !x.Num().IsInt() {
		return e
	}
	z := &big.Int{} // Zero
	a := &Exp{terms: make(map[string]Term)}
	for s, v := range e.terms {
		if !v.Coeff.IsInt() {
			a.terms[s] = v
			continue
		}
		t := big.NewInt(1)
		u := big.NewInt(1)
		_, d := t.DivMod(v.Coeff.Num(), x.Num().Num(), u)
		if d.Cmp(z) == 0 {
			// Drop this term since it is a multiple of x.
			continue
		}
		r := &big.Rat{}
		r.SetInt(u)
		a.terms[s] = Term{
			Coeff: r,
			Fact:  v.Fact,
		}
	}
	return a
}

// Mul computes the product of a series of expressions.
func Mul(as ...*Exp) *Exp {
	var e *Exp
	for i, a := range as {
		if i == 0 {
			e = Sum(a)
			continue
		}
		f := &Exp{
			terms: make(map[string]Term),
		}
		for _, p := range a.terms {
			for _, q := range e.terms {
				x := []factor.Value{factor.R(p.Coeff), factor.R(q.Coeff)}
				n, fs, s := factor.Segment(append(x, append(p.Fact, q.Fact...)...)...)
				f.insert(n, fs, s)
			}
		}
		e = f
	}
	return e
}

// Mul computes the product of this expression with some others.
func (e *Exp) Mul(es ...*Exp) *Exp {
	return Mul(append([]*Exp{e}, es...)...)
}

// Substitute replaces each occurrence of b in an expression with the
// expression c.
func (e *Exp) Substitute(b []factor.Value, c *Exp) *Exp {
	s := [][]factor.Value{}
	for _, t := range c.terms {
		s = append(s, append([]factor.Value{factor.R(t.Coeff)}, t.Fact...))
	}
	g := e
	for {
		again := false
		f := &Exp{
			terms: make(map[string]Term),
		}
		for _, x := range g.terms {
			a := append([]factor.Value{factor.R(x.Coeff)}, x.Fact...)
			hit, y := factor.Replace(a, b, zero, 1)
			if hit == 0 {
				n, fs, tag := factor.Segment(y...)
				f.insert(n, fs, tag)
				// If nothing substituted, then only insert once.
				continue
			}
			if len(s) == 0 {
				// If we are substituting 0 then we won't need anything.
				continue
			}
			again = true
			for _, t := range s {
				_, y := factor.Replace(a, b, t, 1)
				n, fs, tag := factor.Segment(y...)
				f.insert(n, fs, tag)
			}
		}
		g = f
		if !again {
			break
		}
	}
	return g
}

// Contains investigates an expression for the presence of a term, b.
func (e *Exp) Contains(b []factor.Value) bool {
	for _, x := range e.terms {
		a := append([]factor.Value{factor.R(x.Coeff)}, x.Fact...)
		if hit, _ := factor.Replace(a, b, zero, 1); hit != 0 {
			return true
		}
	}
	return false
}

// AsNumber ignores all terms that contain symbols, and just returns
// the value of the constant term. The returned boolean is true only
// when there are no non-constant terms.
func (e *Exp) AsNumber() (*big.Rat, bool) {
	ok := e == nil
	if !ok {
		for _, t := range e.terms {
			if len(t.Fact) == 0 {
				return t.Coeff, len(e.terms) == 1
			}
		}
	}
	return zero[0].Num(), ok
}

// Terms returns the prevailing coefficient and array of unsorted
// simplified terms associated with an expression.
func (e *Exp) Terms() map[string]Term {
	if e == nil {
		return nil
	}
	return e.terms
}

// Common returns the factors common to all terms in the supplied
// expressions as.
func Common(as ...*Exp) Term {
	var f []factor.Value
done:
	for _, a := range as {
		for _, t := range a.terms {
			if f == nil {
				for _, x := range t.Fact {
					f = append(f, x)
				}
				continue
			}
			f = factor.GCF(f, t.Fact)
			if f == nil {
				break done
			}
		}
	}
	return Term{
		Coeff: big.NewRat(1, 1),
		Fact:  f,
	}
}

// ParseExp parses an expression in the form of a string. Only simple
// expressions are parsed: nothing involving parentheses.
func ParseExp(s string) (*Exp, error) {
	e := NewExp()
	for i := 0; i < len(s); {
		vs, d, err := factor.Parse(s[i:])
		i += d
		switch err {
		case factor.ErrSyntax:
			return nil, err
		case factor.ErrDone:
			if i != len(s) && len(vs) == 0 {
				return nil, fmt.Errorf("%q: %v", s[i:], factor.ErrSyntax)
			}
		case nil:
		default:
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		e = e.Add(NewExp(vs))
		if i != len(s) && s[i] == '+' {
			i++
			if i == len(s) {
				return nil, factor.ErrSyntax
			}
		}
	}
	return e, nil
}

// Equals compares two expressions and determines if they are always
// equal.
func (e *Exp) Equals(x *Exp) bool {
	return e.Sub(x).IsZero()
}

// Symbols returns a sorted array of unique symbols found in an
// expression. The returned array should be considered a list and not
// a meaninful product of factors.
func (e *Exp) Symbols() (syms []factor.Value) {
	ss := make(map[string]bool)
	for _, t := range e.terms {
		for _, v := range t.Fact {
			if s := v.Symbol(); s != "" && !ss[s] {
				ss[s] = true
				syms = append(syms, factor.S(s))
			}
		}
	}
	sort.Sort(factor.ByAlpha(syms))
	return
}
