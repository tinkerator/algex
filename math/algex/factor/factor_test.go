package factor

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	vs := []struct {
		v Value
		s string
	}{
		{v: D(1, 1), s: "1"},
		{v: D(0, 1), s: "0"},
		{v: D(-3, 1), s: "-3"},
		{v: S("x"), s: "x"},
		{v: Sp("a", 0), s: "1"},
		{v: Sp("w", -2), s: "w^-2"},
	}
	for i, x := range vs {
		if s := x.v.String(); s != x.s {
			t.Errorf("[%d] got=%q want=%q", i, s, x.s)
		}
	}
}

func TestSimplify(t *testing.T) {
	vs := []struct {
		v []Value
		s string
		c string
	}{
		{
			v: []Value{},
			s: "0",
			c: "0",
		},
		{
			v: []Value{S("x"), S("y"), D(1, 3), S("a")},
			s: "x*y*1/3*a",
			c: "1/3*a*x*y",
		},
		{
			v: []Value{D(3, 1), S("y"), D(1, 3), S("a")},
			s: "3*y*1/3*a",
			c: "a*y",
		},
		{
			v: []Value{D(3, 1), D(-1, 6), S("a"), D(2, 1)},
			s: "3*-1/6*a*2",
			c: "-a",
		},
		{
			v: []Value{D(3, 1), D(-1, 6), S("a"), D(2, 1), Sp("a", 2)},
			s: "3*-1/6*a*2*a^2",
			c: "-a^3",
		},
		{
			v: []Value{D(3, 1), S("a"), S("a"), D(1, 3), Sp("a", -2)},
			s: "3*a*a*1/3*a^-2",
			c: "1",
		},
		{
			v: []Value{D(3, 1), Sp("a", -1), S("a"), D(1, 3), Sp("a", 0)},
			s: "3*a^-1*a*1/3*1",
			c: "1",
		},
	}
	for i, x := range vs {
		if s := Prod(x.v...); s != x.s {
			t.Errorf("[%d] got=%q want=%q", i, s, x.s)
		}
		if c := Prod(Simplify(x.v...)...); c != x.c {
			t.Errorf("[%d] got=%q want=%q", i, c, x.c)
		}
	}
}

func TestReplace(t *testing.T) {
	vs := []struct {
		a, b, c []Value
		n       int
		s       string
	}{
		{
			a: []Value{Sp("a", 3)},
			b: []Value{S("a")},
			c: []Value{S("b")},
			n: 3,
			s: "b^3",
		},
		{
			a: []Value{Sp("a", 3)},
			b: []Value{Sp("a", 2)},
			c: []Value{S("b")},
			n: 1,
			s: "a*b",
		},
		{
			a: []Value{Sp("a", -3)},
			b: []Value{S("a")},
			c: []Value{S("b")},
			n: 0,
			s: "a^-3",
		},
		{
			a: []Value{Sp("a", -3), Sp("b", -3)},
			b: []Value{Sp("b", -2)},
			c: []Value{S("b")},
			n: 1,
			s: "a^-3",
		},
		{
			a: []Value{Sp("a", -3), Sp("b", -3)},
			b: []Value{Sp("a", -1), Sp("b", -2)},
			c: []Value{S("a")},
			n: 1,
			s: "a^-1*b^-1",
		},
	}
	for i, v := range vs {
		if n, x := Replace(v.a, v.b, v.c, 0); n != v.n {
			t.Errorf("[%d] expected %d replacements: got %d", i, v.n, n)
		} else if s := Prod(x...); s != v.s {
			t.Errorf("[%d] got=%q want=%q", i, s, v.s)
		}
	}
}

func TestParse(t *testing.T) {
	vs := []struct {
		before, after, trimmed string
	}{
		{"1", "1", "1"},
		{"0", "0", "0"},
		{"1*1", "1", "1"},
		{"-a", "-1*a", "-a"},
		{"- x^2", "-1*x^2", "-x^2"},
		{"a", "a", "a"},
		{"-a/ 2", "-1/2*a", "-1/2*a"},
		{"a1 ^3* 6 * a1 ^ -2", "6*a1", "6*a1"},
		{"a1^3*6*a1^- 2", "6*a1", "6*a1"},
		{"2^6", "64", "64"},
		{"a^-2*2^-3", "1/8*a^-2", "1/8*a^-2"},
		{"a/b^2", "a*b^-2", "a*b^-2"},
		{"a/-b^2", "-1*a*b^-2", "-a*b^-2"},
	}
	for i, v := range vs {
		x, j, err := Parse(v.before)
		if j != len(v.before) {
			t.Errorf("[%d] wrong length: got=%d, want=%d", i, j, len(v.before))
		}
		if err != nil {
			t.Errorf("[%d] parsing %q yielded: %v", i, v.before, err)
		}
		var xs []string
		for _, s := range x {
			u := s.String()
			if len(x) == 1 || u != "1" {
				xs = append(xs, s.String())
			}
		}
		// First verify all the constituent factors.
		text := "0"
		if len(xs) != 0 {
			text = strings.Join(xs, "*")
		}
		if text != v.after {
			t.Errorf("[%d] test %q -> %v got=%q (want %q)", i, v.before, x, text, v.after)
		}
		// Next verify the condensed representation (+1 & -1 collapsed).
		if text = Prod(x...); text != v.trimmed {
			t.Errorf("[%d] test %q -> %v got=%q (want %q)", i, v.before, x, text, v.trimmed)
		}
	}
}

func TestGCF(t *testing.T) {
	vs := []struct {
		a, b []Value
		s    string
	}{
		{
			a: []Value{Sp("a", 3), Sp("b", 2)},
			b: []Value{Sp("a", 1), Sp("b", 2)},
			s: "a*b^2",
		},
		{
			a: []Value{Sp("a", 3), Sp("b", 2)},
			b: []Value{Sp("a", 4), Sp("b", 1)},
			s: "a^3*b",
		},
		{
			a: []Value{Sp("a", 3), Sp("b", 2)},
			b: []Value{Sp("a", 4), Sp("b", 3)},
			s: "a^3*b^2",
		},
		{
			a: []Value{Sp("a", 3), Sp("b", 2), Sp("c", 2)},
			b: []Value{Sp("a", 4), Sp("c", 3)},
			s: "a^3*c^2",
		},
	}
	for i, v := range vs {
		g := GCF(v.a, v.b)
		if s := Prod(g...); s != v.s {
			t.Errorf("[%d] got=%q want=%q", i, s, v.s)
		}
	}
}

func TestLCP(t *testing.T) {
	vs := []struct {
		a, b []Value
		s    string
	}{
		{
			a: []Value{Sp("a", 3), Sp("b", 2)},
			b: []Value{Sp("a", 1), Sp("b", 2)},
			s: "a^3*b^2",
		},
		{
			a: []Value{Sp("a", 3), Sp("b", 2)},
			b: []Value{Sp("a", 4), Sp("b", 1)},
			s: "a^4*b^2",
		},
		{
			a: []Value{Sp("a", 3), Sp("b", 2)},
			b: []Value{Sp("a", 4), Sp("b", 3)},
			s: "a^4*b^3",
		},
		{
			a: []Value{Sp("a", 3), Sp("b", 2), Sp("c", 2)},
			b: []Value{Sp("a", 4), Sp("c", 3)},
			s: "a^4*b^2*c^3",
		},
	}
	for i, v := range vs {
		g := LCP(v.a, v.b)
		if s := Prod(g...); s != v.s {
			t.Errorf("[%d] got=%q want=%q", i, s, v.s)
		}
	}
}

func TestDen(t *testing.T) {
	vs := []struct {
		a, b []Value
		s    string
	}{
		{
			a: []Value{Sp("a", 3), Sp("b", 2)},
			s: "0",
		},
		{
			a: []Value{Sp("a", -3), Sp("b", 2)},
			s: "a^3",
		},
		{
			a: []Value{Sp("a", -3), Sp("b", -2)},
			s: "a^3*b^2",
		},
		{
			a: []Value{Sp("a", 3), Sp("b", -2), Sp("c", -2)},
			s: "b^2*c^2",
		},
	}
	for i, v := range vs {
		g := Den(v.a)
		if s := Prod(g...); s != v.s {
			t.Errorf("[%d] got=%q want=%q", i, s, v.s)
		}
	}
}
