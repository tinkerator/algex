// Package factor defines basic factors: rationals and symbols.
package factor

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Value captures a single factor. It is either a number or a symbol.
type Value struct {
	num *big.Rat

	pow int
	sym string
}

// IsNum indicates that v is a rational number.
func (v Value) IsNum() bool {
	return v.num != nil
}

// Num simply returns the num value of the term.
func (v Value) Num() *big.Rat {
	return v.num
}

// String displays a single factor.
func (v Value) String() string {
	if v.num != nil {
		return v.num.RatString()
	}
	if v.sym != "" {
		if v.pow == 1 {
			return v.sym
		}
		return fmt.Sprintf("%s^%d", v.sym, v.pow)
	}
	return "<ERROR>"
}

// Symbol returns the symbol associated with this value or "" if no
// symbol is present in v.
func (v Value) Symbol() string {
	return v.sym
}

// zero is a constant zero for comparisons.
var zero = big.NewRat(0, 1)

// one is a constant one for comparisons.
var one = big.NewRat(1, 1)

// minusOne is a constant -one for comparisons.
var minusOne = big.NewRat(-1, 1)

// R copies a rational value into a number value.
func R(n *big.Rat) Value {
	c := new(big.Rat)
	return Value{num: c.Set(n)}
}

// I copies an integer value into a number value.
func I(n *big.Int) Value {
	c := new(big.Rat)
	return Value{num: c.SetInt(n)}
}

// D converts two integers to a rational number value.
func D(num, den int64) Value {
	return Value{num: big.NewRat(num, den)}
}

// S converts a string into a symbol value.
func S(sym string) Value {
	return Value{sym: sym, pow: 1}
}

// Sp converts a string, power to a symbol value.
func Sp(sym string, pow int) Value {
	if pow == 0 {
		return D(1, 1)
	}
	return Value{sym: sym, pow: pow}
}

type ByAlpha []Value

func (a ByAlpha) Len() int      { return len(a) }
func (a ByAlpha) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAlpha) Less(i, j int) bool {
	if a[i].sym < a[j].sym {
		return true
	}
	if a[i].sym > a[j].sym {
		return false
	}
	// Higher powers first (after simplify this is moot).
	return a[i].pow > a[j].pow
}

// Simplify condenses an unsorted array (product) of values into a
// simplified (ordered) form.
func Simplify(vs ...Value) []Value {
	if len(vs) == 0 {
		return nil
	}

	var syms []Value
	n := big.NewRat(1, 1)
	for _, v := range vs {
		if v.num != nil {
			if zero.Cmp(v.num) == 0 {
				return nil
			}
			n.Mul(n, v.num)
			continue
		}
		syms = append(syms, v)
	}
	sort.Sort(ByAlpha(syms))

	res := []Value{R(n)}
	for _, s := range syms {
		i := len(res) - 1
		last := res[i]
		if last.sym != s.sym {
			res = append(res, s)
			continue
		}
		last.pow += s.pow
		if last.pow == 0 {
			res = res[:i]
			continue
		}
		res = append(res[:i], last)
	}
	return res
}

// Prod returns a string representing a product of values. This
// function does not attempt to simplify the array first.
func Prod(vs ...Value) string {
	if len(vs) == 0 {
		return "0"
	}
	var x []string
	prefix := ""
	for i, v := range vs {
		if v.num != nil && i == 0 && len(vs) != 1 {
			if one.Cmp(v.num) == 0 {
				continue
			}
			if minusOne.Cmp(v.num) == 0 {
				prefix = "-"
				continue
			}
		}
		x = append(x, v.String())
	}
	return prefix + strings.Join(x, "*")
}

// Segment simplifies a set of factors and returns the numerical
// coefficient, the non-numeric array of factors and a string
// representation of this array of non-numeric factors.
func Segment(vs ...Value) (*big.Rat, []Value, string) {
	x := Simplify(vs...)
	if len(x) == 0 {
		return nil, nil, ""
	}
	return x[0].num, x[1:], Prod(x[1:]...)
}

// Order returns the power complexity of the Value slice, a.
func Order(a []Value) int {
	n := 0
	for _, v := range a {
		n += v.pow
	}
	return n
}

// Replace replaces copies of b found in a with c. The number of times b
// appeared in a is returned as well as the replaced array of factors.
func Replace(a, b, c []Value, max int) (int, []Value) {
	pn, pf, _ := Segment(b...)
	qf := Simplify(a...)
	r := pn.Inv(pn)
	n := 0
	for len(pf) > 0 && (max <= 0 || n < max) {
		var nf []Value
		i := 0
		j := 0
	GIVEUP:
		for i < len(pf) && j < len(qf) {
			t := pf[i]
			for j < len(qf) {
				u := qf[j]
				j++
				if u.num != nil || t.sym != u.sym {
					nf = append(nf, u)
					continue
				}
				if t.pow*u.pow < 0 {
					// Same symbol, but we require that
					// the sign of the power is the same.
					break GIVEUP
				}
				np := u.pow - t.pow
				if np*t.pow < 0 {
					break GIVEUP
				}
				if np != 0 {
					nf = append(nf, Sp(t.sym, np))
				}
				i++
				break
			}
		}
		if i != len(pf) {
			break
		}
		// Whole match found.
		qf = Simplify(append(append(nf, qf[j:]...), append(c, R(r))...)...)
		n++
	}
	return n, qf
}

// GCF returns the greatest common factor of a set of symbolic terms.
func GCF(a, b []Value) []Value {
	var g []Value
	for i, j := 0, 0; i < len(a) && j < len(b); {
		x, y := a[i], b[j]
		if x.sym < y.sym {
			i++
		} else if x.sym > y.sym {
			j++
		} else {
			i++
			j++
			if x.pow < y.pow {
				g = append(g, x)
			} else {
				g = append(g, y)
			}
		}
	}
	return g
}

// Inv generates the inverse of a set of symbolic factors. It ignores
// Values with no symbol.
func Inv(a []Value) []Value {
	var r []Value
	for _, x := range a {
		if x.pow == 0 {
			continue
		}
		r = append(r, Value{
			pow: -x.pow,
			sym: x.sym,
		})
	}
	return r
}

// Den picks all of the negative power symbols from a list of values
// and inverts them.
func Den(vs []Value) []Value {
	var r []Value
	for _, x := range vs {
		if x.pow >= 0 {
			continue
		}
		r = append(r, Value{
			pow: -x.pow,
			sym: x.sym,
		})
	}
	return r
}

// LCP returns the least common product of a set of symbolic terms.
func LCP(a, b []Value) []Value {
	g := Inv(GCF(a, b))
	return Simplify(append(append(g, a...), b...)...)
}

const allLetters = "abcdefghijklmnopqrstuvwxyz_"
const allDigits = "0123456789"

var ErrDone = errors.New("factor parsing done")
var ErrSyntax = errors.New("syntax problem")

// skipSpace count the number of prefix spaces.
func skipSpace(s string) int {
	for i := 0; i < len(s); i++ {
		if !strings.Contains(" \t\n\r", s[i:i+1]) {
			return i
		}
	}
	return 0
}

var isValidLabel = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`).MatchString

// ValidSymbol confirms that a symbol can be considered externally
// meaningful. Various packages may use other forms for book keeping
// purposes (factoring etc), but for "external" purposes this is the
// only valid form.
func ValidSymbol(token string) bool {
	return isValidLabel(token)
}

// subParse parses the next token of interest or returns ErrDone.
// Leading spaces are ignored. When a leading plus or minus sign is
// considered part of the factors, the signOK value is true. Otherwise
// a plus or minus sign will cause subParse to return ErrDone.
func subParse(signOK bool, s string) (string, int, error) {
	base := skipSpace(s)
	if base == len(s) {
		return "", 0, ErrDone
	}
	if strings.Contains("^*/", s[base:base+1]) {
		return s[base : base+1], base + 1, nil
	}
	sign := ""
	if c := s[base : base+1]; strings.Contains("+-", c) {
		if !signOK {
			return "", base, ErrDone
		}
		sign = c
		base += 1 + skipSpace(s[base+1:])
	}
	if strings.Contains(allDigits, s[base:base+1]) {
		for i := 1 + base; i < len(s); i++ {
			if !strings.Contains(allDigits, s[i:i+1]) {
				return sign + s[base:i], i, nil
			}
		}
		// Everything is the signed number.
		return sign + s[base:], len(s), nil
	}
	if sign != "" {
		return sign, base, nil
	}
	for i := base; i < len(s); i++ {
		if strings.Contains(allLetters, strings.ToLower(s[i:i+1])) {
			continue
		}
		if i > base && strings.Contains(allDigits, s[i:i+1]) {
			continue
		}
		if i == base {
			return "", 0, ErrDone
		}
		return s[base:i], i, nil
	}
	return s[base:], len(s), nil
}

const (
	parseNone = iota
	parseMul
	parsePow
	parseDiv
)

// Parse parses a simple list of string arguments into an
// product of factors. Supported combinations are:
//
//	-33*y*x  -> -33*x*y
//	+33*x^4*y^-3*z/x/3 -> 11*x^3*y^-3*z
func Parse(s string) ([]Value, int, error) {
	modifier := parseMul
	signOK := true
	var vs []Value
	var i int
	for i < len(s) {
		tok, d, err := subParse(signOK, s[i:])
		// fmt.Printf("[%s] %v %d %q %d: %v\n", s[i:], signOK, modifier, tok, d, err)
		if err != nil {
			return Simplify(vs...), i, err
		}
		if strings.Contains(allLetters, strings.ToLower(tok[:1])) {
			switch modifier {
			case parsePow, parseNone:
				return nil, 0, ErrSyntax
			case parseMul:
				vs = append(vs, S(tok))
			case parseDiv:
				vs = append(vs, Sp(tok, -1))
			}
			modifier = parseNone
			signOK = false
		} else if strings.Contains("+-"+allDigits, tok[:1]) {
			if tok == "+" || tok == "-" {
				if !signOK {
					return Simplify(vs...), i, ErrDone
				}
				if tok == "-" {
					vs = append([]Value{D(-1, 1)}, vs...)
				}
				i += d
				continue
			}
			switch modifier {
			case parsePow:
				n, err := strconv.Atoi(tok)
				if err != nil {
					return nil, 0, ErrSyntax
				}
				z := vs[len(vs)-1].num
				if z == nil {
					vs[len(vs)-1].pow *= n
					break
				}
				neg := n < 0
				if neg {
					n = -n
				}
				en := big.NewInt(int64(n))
				num := new(big.Rat).SetInt(new(big.Int).Exp(z.Num(), en, nil))
				den := new(big.Rat).SetInt(new(big.Int).Exp(z.Denom(), en, nil))
				q := new(big.Rat)
				if neg {
					q.Inv(num)
					q.Mul(q, den)
				} else {
					q.Inv(den)
					q.Mul(q, num)
				}
				vs[len(vs)-1].num = q
			case parseNone:
				return nil, 0, ErrSyntax
			case parseMul:
				num, ok := new(big.Rat).SetString(tok)
				if !ok {
					return nil, 0, ErrSyntax
				}
				vs = append(vs, Value{
					num: num,
				})
			case parseDiv:
				num, ok := new(big.Rat).SetString("1/" + tok)
				if !ok {
					return nil, 0, ErrSyntax
				}
				vs = append(vs, Value{
					num: num,
				})
			}
			modifier = parseNone
			signOK = false
		} else {
			if modifier != parseNone {
				return nil, 0, ErrSyntax
			}
			signOK = true
			switch tok[0] {
			case '^':
				modifier = parsePow
			case '*':
				modifier = parseMul
			case '/':
				modifier = parseDiv
			default:
				return Simplify(vs...), i, ErrDone
			}
		}
		i += d
	}
	if modifier != parseNone {
		return nil, 0, ErrSyntax
	}
	return Simplify(vs...), i, nil
}
