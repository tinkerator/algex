// Program ik explores the algebraic properties of a 6 rotational axis
// robot. It attempts to reduce the complexity of the various
// equations relating a desired pose to individual joint rotations. In
// this way, the output can be used to solve the inverse kinematics of
// such a robot.
package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"

	"zappem.net/math/algex/factor"
	"zappem.net/math/algex/matrix"
	"zappem.net/math/algex/rotation"
	"zappem.net/math/algex/terms"
)

var (
	debug = flag.Bool("debug", false, "more verbose output")
)

const zero = "0"

var one = terms.NewExp([]factor.Value{factor.D(1, 1)})
var minusOne = terms.NewExp([]factor.Value{factor.D(-1, 1)})

func vec(x ...string) *matrix.Matrix {
	v, _ := matrix.NewMatrix(3, 1)
	for i, c := range x {
		if c == zero {
			continue
		}
		y, err := terms.ParseExp(c)
		if err != nil {
			log.Fatalf("failed to compute vector [%d]: %v", i, err)
		}
		v.Set(i, 0, y)
	}
	return v
}

type cs []string

func (xs cs) Len() int {
	return len(xs)
}

func (xs cs) Less(i, j int) bool {
	a, b := xs[i], xs[j]
	if len(a) < len(b) {
		return true
	}
	if len(a) > len(b) {
		return false
	}
	return a < b
}

func (xs cs) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

type cleaner struct {
	b   []factor.Value
	e   *terms.Exp
	div *terms.Exp
}

// As we develop the equations, we also develop ways to reduce them.
type cleaners []cleaner

func (xs cleaners) Len() int {
	return len(xs)
}

func (xs cleaners) Less(i, j int) bool {
	a, b := fmt.Sprint(xs[i].b), fmt.Sprint(xs[j].b)
	if len(a) < len(b) {
		return true
	}
	if len(a) > len(b) {
		return false
	}
	return a < b
}

func (xs cleaners) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

// applyIdentities performs some simplifications.
func applyIdentities(e *terms.Exp) *terms.Exp {
	f1, _, _ := factor.Parse("c1*c2")
	e1, _ := terms.ParseExp("c12+s1*s2")
	f2, _, _ := factor.Parse("c1*s2")
	e2, _ := terms.ParseExp("s12-s1*c2")
	return e.Substitute(f1, e1).Substitute(f2, e2)
}

// splitUp takes an expression known to evaluate to zero and extracts
// the most complicated term and generates an expression for it. For
// complexity, we are only interested in factors with c or s as their
// starting letter. All other symbols are considered known constants.
func splitUp(e *terms.Exp) cleaner {
	easy := applyIdentities(e)

	ts := easy.Terms()
	max := 0
	maxK := ""
	var maxFS []factor.Value
	for k, t := range ts {
		n := 0
		var fs []factor.Value
		var cst []factor.Value
		for _, ts := range t.Fact {
			s := ts.String()
			if strings.Contains("cs", s[:1]) {
				n += len(s)
				fs = append(fs, ts)
			} else {
				cst = append(cst, ts)
			}
		}
		if n < max {
			continue
		}
		if n == max && k > maxK {
			continue
		}
		max = n
		maxK = k
		maxFS = fs // sin/cos factors
	}

	// Divide expression into maxFS * N * e1 = - e2.

	var one = []factor.Value{factor.R(big.NewRat(1, 1))}

	rhs := terms.NewExp()
	div := terms.NewExp()

	if len(maxFS) != 0 {
		for _, t := range ts {
			a := append([]factor.Value{factor.R(t.Coeff)}, t.Fact...)
			hit, b := factor.Replace(a, maxFS, one, 1)
			if hit == 0 {
				rhs = rhs.Sub(terms.NewExp(a))
				continue
			}
			div = div.Add(terms.NewExp(b))
		}
	}

	// factor out any negative sign.
	if div.String()[0] == '-' {
		div = terms.Mul(div, minusOne)
		rhs = terms.Mul(rhs, minusOne)
	}

	// Does div have any common multiples with rhs? If so, we
	// factor them out, and let them cancel. Note, when doing
	// this we even factor out negative powers - this helps keep
	// factors in the numerator.
	c := terms.Common(div, rhs)
	if c.Fact != nil {
		r := terms.NewExp(factor.Inv(c.Fact))
		rhs = terms.Mul(r, rhs)
		div = terms.Mul(r, div)
	}

	return cleaner{
		b:   maxFS,
		e:   rhs,
		div: div,
	}
}

// Display a simple refactoring of the cleaner.
func showCleaner(prefix string, r cleaner) {
	if c := terms.Common(r.e); c.Fact != nil {
		n := terms.NewExp(factor.Inv(c.Fact))
		fmt.Print(prefix, r.b, " = (", terms.NewExp(c.Fact), ") * (", terms.Mul(n, r.e))
	} else {
		fmt.Print(prefix, r.b, " = (", r.e)
	}
	fmt.Println(") / (", r.div, ")")
}

// solve takes a bunch of simultaneous equations for the same quantity
// and tries to solve the equations for them recursively. The theory
// of execution is that we ignore the lhs of each expression and
// equate pairs of the remaining equations - the first of them with
// all the others. The n-1 of them are simultaneous equations for the
// n-1 quantities.
// The hope is if we call solve with this lesser set, we may yield
// solutions to individual quantities.
func solve(prefix string, rs cleaners) {
	prefix = prefix + "..."

	if *debug {
		fmt.Println(prefix, "solve", len(rs))
	}

	sort.Sort(rs)

	base := 0
	baseB := fmt.Sprint(rs[0].b)

	for i := 1; i < len(rs); i++ {
		r := rs[i]
		b := fmt.Sprint(r.b)
		if b == baseB {
			if i+1 != len(rs) {
				showCleaner(prefix, r)
				continue
			}
			i++
		}
		if i-base > 1 {
			if *debug {
				fmt.Println(prefix, "sub-solving for", rs[base].b, base, i)
			}
			var roll cleaners
			for j := base + 1; j < i; j++ {
				z := terms.Mul(rs[base].e, rs[j].div).Sub(terms.Mul(rs[j].e, rs[base].div))
				r := splitUp(z)
				if !r.e.IsZero() {
					showCleaner(prefix, r)
					roll = append(roll, r)
				}
			}
			if len(roll) > 1 {
				solve(prefix, roll)
			} else {
				// Too much redundancy, so just
				// display one of the terms.
				showCleaner(prefix, r)
			}
		} else {
			// maximally reduced term.
			showCleaner(prefix, r)
		}
		base = i
		baseB = b
	}
}

func main() {
	axis := []struct {
		j *matrix.Matrix
		v *matrix.Matrix
	}{
		{j: rotation.RZ("0"), v: vec(zero, zero, "d0")},
		{j: rotation.RX("1"), v: vec(zero, zero, "d1")},
		{j: rotation.RX("2"), v: vec(zero, zero, "d2")},
		{j: rotation.RZ("3"), v: vec(zero, zero, "d3")},
		{j: rotation.RX("4"), v: vec(zero, zero, "d4")},
		{j: rotation.RZ("5"), v: vec(zero, zero, zero)},
	}

	// Desired pose.
	r := vec("r0", "r1", "r2")
	b, _ := matrix.NewMatrix(3, 3)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			b.Set(i, j, terms.NewExp([]factor.Value{
				factor.S(fmt.Sprintf("B%d%d", i, j)),
			}))
		}
	}

	fmt.Printf("r = %v\n", r)
	fmt.Printf("b = %v\n", b)

	// These arrays will hold the 6 matrix equations.
	var lhsRs, lhsBs, rhsRs, rhsBs []*matrix.Matrix

	// The next bit of code, could be done with one loop, but in
	// the interests of readability, we redo a lot of work to
	// build the matrix equations.
	for i, a := range axis {
		fmt.Printf("a[%d] = %v, %v\n", i, a.j, a.v)

		// i = number of matrices we pre-multiply by.
		lhsR := r
		lhsB := b
		var err error
		for j := 0; j < i; j++ {
			t := axis[j].j.Transpose()
			lhsR, err = t.Mul(lhsR)
			if err != nil {
				log.Fatalf("i=%d,j=%d: lhsR failure: %v", i, j, err)
			}
			lhsR = lhsR.Add(axis[j].v, minusOne)
			lhsB, err = t.Mul(lhsB)
			if err != nil {
				log.Fatalf("i=%d,j=%d: lhsB failure: %v", i, j, err)
			}
		}
		lhsRs = append(lhsRs, lhsR)
		lhsBs = append(lhsBs, lhsB)

		rhsR, err := matrix.NewMatrix(3, 1)
		if err != nil {
			log.Fatalf("i=%d: rhsR creation failure: %v", i, err)
		}
		rhsB, _ := matrix.Identity(3)
		for j := len(axis) - 1; j >= i; j-- {
			d := rhsR.Add(axis[j].v, one)
			rhsR, err = axis[j].j.Mul(d)
			if err != nil {
				log.Fatalf("i=%d,j=%d: rhsR failure: %v", i, j, err)
			}
			rhsB, err = axis[j].j.Mul(rhsB)
			if err != nil {
				log.Fatalf("i=%d,j=%d: rhsB failure: %v", i, j, err)
			}
		}
		rhsRs = append(rhsRs, rhsR)
		rhsBs = append(rhsBs, rhsB)
	}

	// Collapse down R and B equations to = 0 expressions */
	var Rs []*matrix.Matrix
	var Bs []*matrix.Matrix
	for i := range lhsRs {
		Rs = append(Rs, lhsRs[i].Add(rhsRs[i], minusOne))
		Bs = append(Bs, lhsBs[i].Add(rhsBs[i], minusOne))
	}

	// Collapse all equations down to a unique set. Remember where
	// they came from in refs for debugging purposes.
	eqs := make(map[string]*terms.Exp)
	refs := make(map[string][]string)

	for k := range Rs {
		for i := 0; i < 3; i++ {
			key := fmt.Sprintf("r[%d:%d]", k, i)
			e := Rs[k].El(i, 0)
			s := e.String()
			if _, ok := eqs[s]; !ok {
				eqs[s] = e
			}
			refs[s] = append(refs[s], key)
			for j := 0; j < 3; j++ {
				e = Bs[k].El(i, j)
				s = e.String()
				if _, ok := eqs[s]; !ok {
					eqs[s] = e
				}
				refs[s] = append(refs[s], fmt.Sprintf("b[%d:%d:%d]", k, i, j))
			}
		}
	}

	var list cs
	for s := range refs {
		list = append(list, s)
	}
	sort.Sort(list)

	var rollers cleaners
	for _, s := range list {
		ts := strings.Join(refs[s], ",")
		if *debug {
			fmt.Println(s, "= 0 :", ts)
		}
		rollers = append(rollers, splitUp(eqs[s]))
	}

	solve("", rollers)
}
