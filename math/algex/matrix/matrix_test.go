package matrix

import (
	"fmt"
	"testing"

	"zappem.net/pub/math/algex/factor"
	"zappem.net/pub/math/algex/terms"
)

func TestNewMatrix(t *testing.T) {
	one, err := Identity(2)
	if err != nil {
		t.Fatalf("failed to make a 2x2 identity matrix!: %v", err)
	}
	if got, want := one.String(), "[[1, 0], [0, 1]]"; got != want {
		t.Errorf("one failed: got=%q, want=%q", got, want)
	}
}

func TestTranspose(t *testing.T) {
	a, err := NewMatrix(2, 3)
	if err != nil {
		t.Fatalf("failed to make 2x3 matrix: %v", err)
	}
	for i := 0; i < a.rows; i++ {
		for j := 0; j < a.cols; j++ {
			a.Set(i, j, terms.NewExp([]factor.Value{factor.D(int64(i+1), 5*int64(j+1))}))
		}
	}
	if got, want := a.String(), "[[1/5, 1/10, 1/15], [2/5, 1/5, 2/15]]"; got != want {
		t.Errorf("failed to make matrix: got=%q, want=%q", got, want)
	}
	b := a.Transpose()
	if got, want := b.String(), "[[1/5, 2/5], [1/10, 1/5], [1/15, 2/15]]"; got != want {
		t.Errorf("failed to transpose matrix: got=%q, want=%q", got, want)
	}
}

func TestMul(t *testing.T) {
	a, err := Identity(2)
	if err != nil {
		t.Fatalf("failed to make 2x2 identity: %v", err)
	}
	b, err := Identity(2)
	if err != nil {
		t.Fatalf("failed to make 2x2 identity: %v", err)
	}
	b.Set(0, 1, terms.Mul(a.El(0, 0), terms.NewExp([]factor.Value{factor.Sp("x", 2)})))

	c, err := a.Mul(b)
	if err != nil {
		t.Fatalf("failed to multiply 2x2 matrices: %v", err)
	}
	if got, want := c.String(), b.String(); got != want {
		t.Errorf("matrix multiply %v*%v: got=%v, want=%v", a, b, got, want)
	}
	d, err := b.Mul(a)
	if err != nil {
		t.Fatalf("failed to multiply 2x2 matrices: %v", err)
	}
	if got, want := d.String(), b.String(); got != want {
		t.Errorf("matrix multiply %v*%v: got=%v, want=%v", a, b, got, want)
	}

	x, err := NewMatrix(2, 3)
	if err != nil {
		t.Fatalf("failed to make 2x3 matrix: %v", err)
	}
	for i := 0; i < x.rows; i++ {
		for j := 0; j < x.cols; j++ {
			v, err := terms.ParseExp(fmt.Sprintf("x%d%d", i, j))
			if err != nil {
				t.Errorf("x[%d][%d] setting failure: %v", i, j, err)
			}
			x.Set(i, j, v)
		}
	}
	y, err := NewMatrix(3, 2)
	if err != nil {
		t.Fatalf("failed to make 3x2 matrix: %v", err)
	}
	for i := 0; i < y.rows; i++ {
		for j := 0; j < y.cols; j++ {
			v, err := terms.ParseExp(fmt.Sprintf("y%d%d", i, j))
			if err != nil {
				t.Errorf("y[%d][%d] setting failure: %v", i, j, err)
			}
			y.Set(i, j, v)
		}
	}
	if z, err := x.Mul(y); err != nil {
		t.Errorf("matrix multiply failure %v*%v: %v", x, y, err)
	} else if got, want := z.String(), "[[x00*y00+x01*y10+x02*y20, x00*y01+x01*y11+x02*y21], [x10*y00+x11*y10+x12*y20, x10*y01+x11*y11+x12*y21]]"; got != want {
		t.Errorf("z: got=%q, want=%q", got, want)
	}
}

func TestSum(t *testing.T) {
	a := terms.NewExp([]factor.Value{factor.D(-2, 3), factor.Sp("x", -1)})
	b := terms.NewExp([]factor.Value{factor.D(9, 4), factor.Sp("x", 2)})

	p, _ := NewMatrix(1, 2)
	p.Set(0, 0, a)
	q, _ := NewMatrix(1, 2)
	q.Set(0, 0, a)
	q.Set(0, 1, b)

	r, err := p.Sum(q, terms.Mul(a, a))
	if err != nil {
		t.Fatalf("can't sum things: %v", err)
	}
	if got, want := r.String(), "[[-2/3*x^-1-8/27*x^-3, 1]]"; got != want {
		t.Errorf("add: got=%q, want=%q", got, want)
	}
}
