// Package terms abstracts sums of products of factors.
package terms

import (
	"errors"
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

// Rat generates an expression of a rational number.
func Rat(r *big.Rat) *Exp {
	return NewExp([]factor.Value{factor.R(r)})
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

// Exp converts a Term into a stand alone expression.
func (term Term) Exp() *Exp {
	e := &Exp{
		terms: make(map[string]Term),
	}
	e.insert(term.Coeff, term.Fact, factor.Prod(term.Fact...))
	return e
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

// Substituted replaces each occurrence of b in an expression with the
// expression c. If the returned boolean is true, then something was
// substituted.
func (e *Exp) Substituted(b []factor.Value, c *Exp) (*Exp, bool) {
	if len(b) == 0 {
		return e, false
	}
	s := [][]factor.Value{}
	for _, t := range c.terms {
		s = append(s, append([]factor.Value{factor.R(t.Coeff)}, t.Fact...))
	}
	g := e
	acted := false
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
		acted = true
	}
	return g, acted
}

// Substitute unconditionally attempts to substitute occurences of b
// in e with expression c. Consider using e.Substituted() to understand
// if any change was made.
func (e *Exp) Substitute(b []factor.Value, c *Exp) *Exp {
	e2, _ := e.Substituted(b, c)
	return e2
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

// Partition splits an expression into two parts: those with a factor
// of b; and those without.
func (e *Exp) Partition(b []factor.Value) (div, rem *Exp) {
	if e == nil {
		return
	}
	d := NewExp()
	r := NewExp()
	for _, x := range e.terms {
		a := append([]factor.Value{factor.R(x.Coeff)}, x.Fact...)
		if hit, _ := factor.Replace(a, b, zero, 1); hit != 0 {
			d = d.Add(NewExp(a))
		} else {
			r = r.Add(NewExp(a))
		}
	}
	if len(d.terms) != 0 {
		div = d
	}
	if len(r.terms) != 0 {
		rem = r
	}
	return
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

// Common returns the non-numerical factors common to all terms in the
// supplied expressions as.
func Common(as ...*Exp) Term {
	var f []factor.Value
	first := true
done:
	for _, a := range as {
		if len(a.terms) == 0 {
			f = nil
			break done
		}
		for _, t := range a.terms {
			if first && f == nil {
				for _, x := range t.Fact {
					f = append(f, x)
				}
				first = false
				continue
			}
			f = factor.GCF(f, t.Fact)
			if f == nil {
				break done
			}
		}
	}
	// TODO investigate factoring out any common factors.
	return Term{
		Coeff: big.NewRat(1, 1),
		Fact:  f,
	}
}

// Frac is used to hold a numerator and a denominator expression.
type Frac struct {
	Num, Den *Exp
}

// String displays a text representation of a ratio.
func (r *Frac) String() string {
	if r == nil || r.Num == nil {
		return "0"
	}
	ns := r.Num.String()
	ds := r.Den.String()
	if ds == "1" {
		return ns
	}
	if _, ok := r.Den.terms["0"]; len(r.Den.terms) != 1 || !ok {
		ds = fmt.Sprint("(", ds, ")")
	}
	if len(r.Num.terms) == 1 {
		return fmt.Sprintf("%s/%s", ns, ds)
	}
	return fmt.Sprintf("(%s)/%s", ns, ds)
}

// NewFrac with no args initializes a ratio value to "0/1". With one
// arg, it returns a fraction with that arg as the numerator and 1 as
// the denominator. For all other numbers of args, the last arg is
// considered the denominator expression, and all of the preceding
// args are summed to construct the numerator.
func NewFrac(e ...*Exp) *Frac {
	switch len(e) {
	case 0:
		return &Frac{
			Den: NewExp([]factor.Value{factor.D(1, 1)}),
		}
	case 1:
		return Ratio(e[0])
	default:
		r := &Frac{
			Num: Sum(e[:len(e)-1]...),
			Den: e[len(e)-1],
		}
		r.Reduce()
		return r
	}
}

// Ratio breaks an expression into a numerator and a denominator
// returned in the form of a Frac.
//
// This function works by searching through each term in e and
// extracts a list of reciprocal (power) terms. It then computes the
// LCF of these terms. This is the denominator. The Numerator is e*d.
func Ratio(e *Exp) (f *Frac) {
	f = NewFrac()

	var den []factor.Value
	for _, ts := range e.terms {
		d := factor.Den(ts.Fact)
		den = factor.LCP(den, d)
	}

	if len(den) != 0 {
		f.Den = NewExp(den)
	}

	f.Num = e.Mul(f.Den)
	return
}

// Substituted replaces each occurrence of b in a Frac numerator and
// denominator with the expression c. The boolean return value is true
// when a substitution was performed.
func (f *Frac) Substituted(b []factor.Value, c *Frac) (*Frac, bool) {
	inv := factor.Inv(b)
	n, d := "_n", "_d"

	num0, a0 := f.Num.Substituted(b, NewExp([]factor.Value{factor.S(n), factor.Sp(d, -1)}))
	num, a1 := num0.Substituted(inv, NewExp([]factor.Value{factor.Sp(n, -1), factor.S(d)}))
	den0, a2 := f.Den.Substituted(b, NewExp([]factor.Value{factor.S(n), factor.Sp(d, -1)}))
	den, a3 := den0.Substituted(inv, NewExp([]factor.Value{factor.Sp(n, -1), factor.S(d)}))

	if !(a0 || a1 || a2 || a3) {
		return f, false // b not found in f.
	}

	r1 := Ratio(num)
	r2 := Ratio(den)
	r := &Frac{
		Num: r1.Num.Mul(r2.Den),
		Den: r1.Den.Mul(r2.Num),
	}

	r.Reduce()
	r.Num = r.Num.Substitute([]factor.Value{factor.S(n)}, c.Num)
	r.Num = r.Num.Substitute([]factor.Value{factor.S(d)}, c.Den)
	r.Den = r.Den.Substitute([]factor.Value{factor.S(n)}, c.Num)
	r.Den = r.Den.Substitute([]factor.Value{factor.S(d)}, c.Den)
	r.Reduce()

	return r, true
}

// Substitute replaces each occurrence of b in a Frac numerator and
// denominator with the expression c. Consider using f.Substituted()
// since this indicates if a substitution occurred.
func (f *Frac) Substitute(b []factor.Value, c *Frac) *Frac {
	f2, _ := f.Substituted(b, c)
	return f2
}

var ErrBadFirstChar = errors.New("invalid first character, \"_\"")

// ParseFrac converts a string into a parsed Frac expression pair.
func ParseFrac(text string) (*Frac, error) {
	// This function uses "_" prefixed values for temporaries,
	// so ban them from being in the supplied string.
	if strings.HasPrefix(text, "_") {
		return nil, ErrBadFirstChar
	}
	return parseFracInt(text)
}

// gcd returns the greatest common divisor of a and b.
func gcd(a, b *big.Int) *big.Int {
	g := big.NewInt(1).GCD(nil, nil, a, b)
	return g.Abs(g)
}

// lcm returns the least common multiple of a and b.
func lcm(a, b *big.Int) *big.Int {
	g := gcd(a, b)
	l := big.NewInt(1).Mul(a, b)
	l = l.Abs(l)
	l = l.Quo(l, g)
	return l
}

// CommonN explores a list of expressions and determines what big.Rat
// can be factored out of all terms. The denominator of this big.Rat
// ensures that the rest of the expression have "1" for denominators,
// and the numerator is common to all of the terms.
func CommonN(exs ...*Exp) *big.Rat {
	once := false
	n := big.NewInt(1)
	d := big.NewInt(1)
	for _, ex := range exs {
		if ex == nil {
			continue
		}
		for _, t := range ex.terms {
			if t.Coeff == nil {
				continue
			}
			d = lcm(d, t.Coeff.Denom())
			if once {
				n = gcd(n, t.Coeff.Num())
				continue
			}
			n = n.Set(t.Coeff.Num())
			once = true
		}
	}
	return big.NewRat(1, 1).SetFrac(n, d)
}

// Leading returns the highest power term from an expression.
func (ex *Exp) Leading() (term Term, err error) {
	// Find the greatest power symbol term of `a`.
	n := 0
	var leading string
	for s, t := range ex.terms {
		m := factor.Order(t.Fact)
		if n > m {
			continue
		}
		if n == m && s > leading {
			continue
		}
		leading = s
		term = t
		n = m
	}
	if n == 0 {
		err = factor.ErrDone
		return
	}
	return
}

// Divide performs long division of one expression with another. It
// returns the quotient: div, and any remainder: rem.
func (ex *Exp) Divide(a *Exp) (div, rem *Exp, err error) {
	lead, err := a.Leading()
	if err != nil {
		return nil, nil, err
	}
	// Express this leading term as _factor-"the rest" of `a`.
	repl := []factor.Value{factor.S("_factor")}
	inv := big.NewRat(1, 1).Inv(lead.Coeff)
	leader := NewExp(append([]factor.Value{factor.R(lead.Coeff)}, lead.Fact...))
	rest := NewExp(repl).Add(leader).Sub(a).Mul(NewExp([]factor.Value{factor.R(inv)}))
	simple := ex.Substitute(lead.Fact, rest)
	x, y := simple.Partition(repl)
	if x == nil {
		return nil, nil, factor.ErrDone
	}
	if common := Common(x, NewExp(repl)); len(common.Fact) != 1 {
		return nil, nil, factor.ErrDone
	}
	div = x.Mul(NewExp(factor.Inv(repl))).Substitute(repl, a)
	return div, y, nil
}

// Reduce removes factors common to the numerator and denominator.
// TODO explore more sophisticated factorization.
func (f *Frac) Reduce() {
	// Reduce the numerical coefficients.
	n := CommonN(f.Num)
	invN := big.NewRat(1, 1).Inv(n)
	d := CommonN(f.Den)
	invD := big.NewRat(1, 1).Inv(d)
	r := big.NewRat(1, 1).Mul(n, invD)
	pN := NewExp([]factor.Value{factor.I(r.Num()), factor.R(invN)})
	pD := NewExp([]factor.Value{factor.I(r.Denom()), factor.R(invD)})
	f.Num = Mul(f.Num, pN)
	f.Den = Mul(f.Den, pD)

	// Reduce simple common factors.
	t := Common(f.Num, f.Den)
	if t.Fact != nil {
		inv := NewExp(factor.Inv(t.Fact))
		f.Num = Mul(f.Num, inv)
		f.Den = Mul(f.Den, inv)
	}

	// Best effort at simplifying the polynomials.

	a, b, err := f.Num.Divide(f.Den)
	if err == nil && b == nil {
		f.Num = a
		f.Den = NewExp([]factor.Value{factor.D(1, 1)})
	}
	a, b, err = f.Den.Divide(f.Num)
	if err == nil && b == nil {
		f.Num = NewExp([]factor.Value{factor.D(1, 1)})
		f.Den = a
	}
}

// parseFracInt implements Frac text parsing on a string that contains
// no externally defined "_" symbols.
func parseFracInt(text string) (r *Frac, err error) {
	depth := 0
	base := -1
	// This loop breaks the text string into X ( Y ) Z pieces,
	// replacing text with something that involves a substitution
	// for X. X does not contain any parentheses. Y and Z may. Y
	// and Z are parsed by recursion.
	subs := make(map[string]*Frac)
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '(' {
			if depth == 0 && base == -1 {
				base = i
			}
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				sub := fmt.Sprintf("_XXX%d", len(subs))
				r2, err2 := ParseFrac(text[base+1 : i])
				if err2 != nil {
					err = err2
					return
				}
				subs[sub] = r2
				// Replace with sub and explore rest.
				text = fmt.Sprintf("%s %s %s", text[:base], sub, text[i+1:])
				i = base + len(sub) - 1
				base = -1
				continue
			}
		}
		if depth == -1 {
			return nil, fmt.Errorf("parsing error too many ')' in %q", text)
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("parsing error too many '(' in %q", text)
	}

	e, err2 := ParseExp(text)
	if err2 != nil {
		return nil, err2
	}
	// Replace each substitution with a numerator and denominator
	// fraction.
	for sub := range subs {
		n, d := sub+"n", sub+"d"
		e = e.Substitute([]factor.Value{factor.S(sub)}, NewExp([]factor.Value{factor.S(n), factor.Sp(d, -1)})).Substitute([]factor.Value{factor.Sp(sub, -1)}, NewExp([]factor.Value{factor.Sp(n, -1), factor.S(d)}))
	}
	r = Ratio(e)
	// Given we have completely separated the positive and
	// negative powers of the ratio components we can now expand
	// the numerator and denominator with simple substitution.
	for sub, val := range subs {
		n, d := fmt.Sprint(sub, "n"), fmt.Sprint(sub, "d")
		r.Num = r.Num.Substitute([]factor.Value{factor.S(n)}, val.Num)
		r.Num = r.Num.Substitute([]factor.Value{factor.S(d)}, val.Den)
		r.Den = r.Den.Substitute([]factor.Value{factor.S(n)}, val.Num)
		r.Den = r.Den.Substitute([]factor.Value{factor.S(d)}, val.Den)
	}
	r.Reduce()
	return
}

// ParseExp parses an expression in the form of a string. Only simple
// expressions are parsed: nothing involving parentheses.
func ParseExp(s string) (*Exp, error) {
	s = strings.TrimRight(s, " \t")
	if len(s) == 0 {
		return nil, factor.ErrSyntax
	}
	e := NewExp()
	for i := 0; i < len(s); {
		vs, d, err := factor.Parse(s[i:])
		switch err {
		case factor.ErrSyntax:
			return nil, fmt.Errorf("%q, %v", s[i:], err)
		case factor.ErrDone:
			if i != len(s) && len(vs) == 0 {
				return nil, fmt.Errorf("%q, %v", s[i:], factor.ErrSyntax)
			}
		case nil:
		default:
			return nil, fmt.Errorf("unexpected error, %q: %v", s[i:], err)
		}
		i += d
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
