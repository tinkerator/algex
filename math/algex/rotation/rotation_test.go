package rotation

import (
	"testing"

	"zappem.net/math/algex/factor"
	"zappem.net/math/algex/matrix"
	"zappem.net/math/algex/terms"
)

func TestR(t *testing.T) {
	for i, r := range []*matrix.Matrix{RX("t"), RY("t"), RZ("t")} {
		r2 := r.Mx(r)
		ans := r2.Substitute(
			[]factor.Value{factor.Sp("ct", 2)},
			terms.NewExp([]factor.Value{factor.S("c2t")},
				[]factor.Value{factor.Sp("st", 2)}),
		).Substitute(
			[]factor.Value{factor.S("ct"), factor.S("st")},
			terms.NewExp([]factor.Value{factor.D(1, 2), factor.S("s2t")}),
		)
		// Should equal r if 2t -> t.
		cf := r.Add(ans, terms.NewExp([]factor.Value{factor.D(-1, 1)})).Substitute(
			[]factor.Value{factor.S("c2t")},
			terms.NewExp([]factor.Value{factor.S("ct")}),
		).Substitute(
			[]factor.Value{factor.S("s2t")},
			terms.NewExp([]factor.Value{factor.S("st")}),
		)
		if cf.String() != "[[0, 0, 0], [0, 0, 0], [0, 0, 0]]" {
			t.Errorf("[%d] got=%v, want=zero", i, cf)
		}
	}
}
