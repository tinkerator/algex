package terms

import (
	"fmt"
	"math/big"
	"testing"

	f "zappem.net/pub/math/algex/factor"
)

func TestNewExp(t *testing.T) {
	vs := []struct {
		e [][]f.Value
		s string
	}{
		{
			e: [][]f.Value{
				{f.D(-3, 1)},
				{f.D(2, 1), f.S("a")},
				{f.D(-4, 1), f.Sp("b", -1)},
			},
			s: "-3+2*a-4*b^-1",
		},
		{
			e: [][]f.Value{
				{f.D(-3, 1)},
				{f.D(2, 1)},
				{f.D(2, 1), f.S("a")},
				{f.D(-4, 1), f.S("a")},
			},
			s: "-1-2*a",
		},
		{
			e: [][]f.Value{
				{f.D(-3, 1)},
				{f.D(3, 1)},
			},
			s: "0",
		},
		{
			e: [][]f.Value{
				{f.D(-3, 1), f.S("a"), f.S("b")},
				{f.D(2, 1), f.Sp("a", 2)},
				{f.D(2, 1), f.S("b"), f.S("a")},
				{f.D(1, 1), f.S("b"), f.S("a")},
			},
			s: "2*a^2",
		},
	}
	for i, v := range vs {
		e := NewExp(v.e...)
		if s := e.String(); s != v.s {
			t.Errorf("[%d] got=%q want=%q", i, s, v.s)
		}
	}
}

func TestAddSub(t *testing.T) {
	a := NewExp([]f.Value{f.Sp("a", 3), f.D(1, 3)},
		[]f.Value{f.D(2, 3), f.Sp("a", 3)},
		[]f.Value{f.Sp("a", -1)})
	b := NewExp([]f.Value{f.Sp("b", 5), f.D(1, 3)},
		[]f.Value{f.Sp("a", -1)})
	vs := []struct {
		e *Exp
		s string
	}{
		{e: a, s: "a^-1+a^3"},
		{e: a.Add(a), s: "2*a^-1+2*a^3"},
		{e: a.Sub(a), s: "0"},
		{e: a.Sub(a.Add(a)), s: "-a^-1-a^3"},
		{e: a.Sub(b), s: "a^3-1/3*b^5"},
	}
	for i, v := range vs {
		if s := v.e.String(); s != v.s {
			t.Errorf("[%d] got=%q want=%q", i, s, v.s)
		}
	}
}

func TestMul(t *testing.T) {
	a := NewExp([]f.Value{f.Sp("a", 3)}, []f.Value{f.Sp("b", 4)})
	b := NewExp([]f.Value{f.Sp("a", 3)}, []f.Value{f.Sp("b", 4), f.D(-1, 1)})
	c := NewExp([]f.Value{f.Sp("a", 6)}, []f.Value{f.Sp("b", 8)})

	vs := []struct {
		e *Exp
		s string
	}{
		{e: Mul(a, a), s: "2*a^3*b^4+a^6+b^8"},
		{e: Mul(b, b), s: "-2*a^3*b^4+a^6+b^8"},
		{e: Mul(a, b), s: "a^6-b^8"},
		{e: Mul(b, a), s: "a^6-b^8"},
		{e: Mul(b, a, c), s: "a^12-b^16"},
	}
	for i, v := range vs {
		if s := v.e.String(); s != v.s {
			t.Errorf("[%d] got=%q want=%q", i, s, v.s)
		}
	}
}

func TestSubstitute(t *testing.T) {
	vs := []struct {
		e, c *Exp
		b    []f.Value
		s    string
	}{
		{
			e: NewExp([]f.Value{f.S("a"), f.S("y")}),
			b: []f.Value{f.S("y")},
			c: NewExp([]f.Value{f.S("a")}, []f.Value{f.D(-1, 1), f.Sp("b", 2), f.Sp("a", -1)}),
			s: "a^2-b^2",
		},
		{
			e: NewExp([]f.Value{f.Sp("a", 2)}),
			b: []f.Value{f.S("a")},
			c: NewExp([]f.Value{f.S("b")}, []f.Value{f.D(-1, 1), f.S("c")}),
			s: "-2*b*c+b^2+c^2",
		},
		{
			e: NewExp([]f.Value{f.Sp("a", 2)}, []f.Value{f.Sp("b", 2)}),
			b: []f.Value{f.S("a")},
			c: NewExp(), // Zero.
			s: "b^2",
		},
	}
	for i, v := range vs {
		r := v.e.Substitute(v.b, v.c)
		if s := r.String(); s != v.s {
			t.Errorf("[%d] %q (%q -> %q) got=%q want=%q", i, v.e, f.Prod(v.b...), v.c, s, v.s)
		}
	}
}

func TestMod(t *testing.T) {
	three := f.D(3, 1)
	for i := int64(-3); i < 7; i++ {
		a := NewExp([]f.Value{f.D(i, 1), f.S("x")})
		e := (i + 9) % 3
		want := "0"
		switch e {
		case 1:
			want = "x"
		case 2:
			want = "2*x"
		}
		if got := a.Mod(three).String(); got != want {
			t.Errorf("[%d] -> a=%q (a mod 3 =) got:%q, want: %q", i, a, a.Mod(three), want)
		}
	}
}

func TestContains(t *testing.T) {
	vs := []struct {
		sym  string
		e    *Exp
		want bool
	}{
		{sym: "x", e: NewExp([]f.Value{f.D(1, 4), f.Sp("x", 3)}), want: true},
		{sym: "y", e: NewExp([]f.Value{f.D(1, 4), f.Sp("x", 3)}), want: false},
	}
	for i, v := range vs {
		if got := v.e.Contains([]f.Value{f.S(v.sym)}); got != v.want {
			t.Errorf("[%d] exp=%q contains %q: got=%v, want=%v", i, v.e, v.sym, got, v.want)
		}
	}
}

func TestParseExp(t *testing.T) {
	vs := []struct {
		from, want string
	}{
		{"a ", "a"},
		{"1+1-1", "1"},
		{"a-b+a", "2*a-b"},
		{"d1", "d1"},
		{"d1*d0", "d0*d1"},
		{"-d1*d1*d0*d1", "-d0*d1^3"},
		{"a+a*b+b*a-a", "2*a*b"},
		{"a+a*b+b*a+a-c/2+2/d", "2*a+2*a*b-1/2*c+2*d^-1"},
	}
	if e, err := ParseExp(" "); err == nil {
		t.Fatalf("parsed empty as something: %v", e)
	}
	for i, v := range vs {
		e, err := ParseExp(v.from)
		if err != nil {
			t.Errorf("[%d] parsing %q: %v", i, v.from, err)
		}
		if got := e.String(); got != v.want {
			t.Errorf("[%d] got=%q want=%q", i, got, v.want)
		}
	}
}

func TestFrac(t *testing.T) {
	ex := []struct{ a, b string }{
		{a: "x ", b: " x"},
		{a: "x+y", b: "y +c+ x -c"},
		{a: "a^2- b*b", b: "- (a+b)*(b-a)"},
		{a: "a/(a+b) + b/(a-b)", b: "(a^2+b^2)/(a^2-b^2)"},
		{a: "al/be", b: "1/(al/be)^-1"},
		{a: "alpha *beta", b: "-beta^2 /-(alpha/beta)^-1"},
	}
	for i, e := range ex {
		a, as, err := ParseFrac(e.a)
		if err != nil {
			t.Errorf("failed for %d:a=%q, a=(%v): %v", i, e.a, a, err)
		}
		if as != nil {
			t.Errorf("array returned, but not expecting one: %v", as)
		}
		b, bs, err := ParseFrac(e.b)
		if err != nil {
			t.Errorf("failed for %d:b=%q, b=(%v): %v", i, e.b, b, err)
		}
		if bs != nil {
			t.Errorf("array returned, but not expecting one: %v", bs)
		}
		if ast, bst := a.String(), b.String(); ast != bst {
			t.Errorf("failed to equate %d:a=%q,b=%q -> %q != %q", i, e.a, e.b, ast, bst)
		}
	}
	ex2 := []struct{ a, b string }{
		{a: "x , y", b: " x,y "},
		{a: "x+y,b", b: "y +c+ x -c ,b"},
	}
	for i, e := range ex2 {
		a, as, err := ParseFrac(e.a)
		if err != nil {
			t.Errorf("failed for %d:a=%q, a=(%v): %v", i, e.a, a, err)
		}
		if a != nil {
			t.Errorf("expect array only, but got one: %v", a)
		}
		b, bs, err := ParseFrac(e.b)
		if err != nil {
			t.Errorf("failed for %d:b=%q, b=(%v): %v", i, e.b, b, err)
		}
		if b != nil {
			t.Errorf("expect array only, but got one: %v", b)
		}
		if ast, bst := fmt.Sprint(as), fmt.Sprint(bs); ast != bst {
			t.Errorf("failed to equate %d:a=%q,b=%q -> %q != %q", i, e.a, e.b, ast, bst)
		}
	}
}

func TestGgcdLcm(t *testing.T) {
	vs := []struct{ a, b, g, l int64 }{
		{1, 2, 1, 2},
		{2, 4, 2, 4},
	}
	for i, v := range vs {
		a := big.NewInt(v.a)
		b := big.NewInt(v.b)
		g := gcd(a, b)
		l := lcm(a, b)
		if want := big.NewInt(v.g); g.Cmp(want) != 0 {
			t.Errorf("[%d] gcd(%v,%v): got=%v want=%v\n", i, a, b, g, want)
		}
		if want := big.NewInt(v.l); l.Cmp(want) != 0 {
			t.Errorf("[%d] lcm(%v,%v): got=%v want=%v\n", i, a, b, l, want)
		}
	}
}
