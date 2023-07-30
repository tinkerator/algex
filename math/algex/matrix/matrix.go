// Package matrix manages matrices of expressions.
package matrix

import (
	"fmt"
	"strings"

	"zappem.net/pub/math/algex/factor"
	"zappem.net/pub/math/algex/terms"
)

type Matrix struct {
	// row count and col count
	rows, cols int
	// The matrix elements arranged, [r=0,c=0], [0,1], [0,2] ...
	data []*terms.Exp
}

// NewMatrix creates a rows x cols matrix.
func NewMatrix(rows, cols int) (*Matrix, error) {
	if rows <= 0 || cols <= 0 {
		return nil, fmt.Errorf("need positive dimensions, not %dx%d", rows, cols)
	}
	m := &Matrix{
		rows: rows,
		cols: cols,
		data: make([]*terms.Exp, rows*cols),
	}
	return m, nil
}

// String serializes a matrix for displaying.
func (m *Matrix) String() string {
	var rs []string
	for r := 0; r < m.rows; r++ {
		var cs []string
		for c := 0; c < m.cols; c++ {
			cs = append(cs, m.data[c+m.cols*r].String())
		}
		rs = append(rs, "["+strings.Join(cs, ", ")+"]")
	}
	return "[" + strings.Join(rs, ", ") + "]"
}

// Set sets the value of a matrix element.
func (m *Matrix) Set(row, col int, e *terms.Exp) error {
	if row < 0 || col < 0 || row >= m.rows || col >= m.cols {
		return fmt.Errorf("bad cell: [%d,%d] in %dx%d matrix", row, col, m.rows, m.cols)
	}
	m.data[col+m.cols*row] = e
	return nil
}

// El returns the row,col element of the matrix.
func (m *Matrix) El(row, col int) *terms.Exp {
	return m.data[col+m.cols*row]
}

// Identity returns a square identity matrix of dimension n.
func Identity(n int) (*Matrix, error) {
	if n <= 0 {
		return nil, fmt.Errorf("invalid identity matrix of dimension n=%d", n)
	}
	m, _ := NewMatrix(n, n)
	for i := 0; i < n; i++ {
		m.Set(i, i, terms.NewExp([]factor.Value{factor.D(1, 1)}))
	}
	return m, nil
}

// Transpose returns the transpose of a specified matrix.
func (m *Matrix) Transpose() *Matrix {
	n, err := NewMatrix(m.cols, m.rows)
	if err != nil {
		panic(err)
	}
	for i := 0; i < m.rows; i++ {
		for j := 0; j < m.cols; j++ {
			n.Set(j, i, m.El(i, j))
		}
	}
	return n
}

// Mul multiplies m x n with conventional matrix multiplication.
func (m *Matrix) Mul(n *Matrix) (*Matrix, error) {
	if m.cols != n.rows {
		return nil, fmt.Errorf("a cols(%d) != b rows(%d)", m.cols, n.rows)
	}
	a, err := NewMatrix(m.rows, n.cols)
	if err != nil {
		return nil, err
	}
	for r := 0; r < a.rows; r++ {
		for c := 0; c < a.cols; c++ {
			var e []*terms.Exp
			for i := 0; i < m.cols; i++ {
				x, y := m.El(r, i), n.El(i, c)
				if x != nil && y != nil {
					e = append(e, terms.Mul(x, y))
				}
			}
			a.Set(r, c, terms.Sum(e...))
		}
	}
	return a, nil
}

// Mx multiplies two matrices and panics on error.
func (m *Matrix) Mx(n *Matrix) *Matrix {
	a, err := m.Mul(n)
	if err != nil {
		panic(err)
	}
	return a
}

// Sum adds two matrices.
func (m *Matrix) Sum(n *Matrix, scale *terms.Exp) (*Matrix, error) {
	if m.rows != n.rows || m.cols != n.cols {
		return nil, fmt.Errorf("inequivalent dimensions %dx%d != %dx%d", m.rows, m.cols, n.rows, n.cols)
	}
	a, _ := NewMatrix(m.rows, m.cols)
	for r := 0; r < m.rows; r++ {
		for c := 0; c < m.cols; c++ {
			if q := n.El(r, c); q == nil {
				a.Set(r, c, m.El(r, c))
			} else if p := m.El(r, c); p == nil {
				a.Set(r, c, terms.Mul(q, scale))
			} else {
				a.Set(r, c, p.Add(terms.Mul(q, scale)))
			}
		}
	}
	return a, nil
}

// Add adds two matrices, and panics on error.
func (m *Matrix) Add(n *Matrix, scale *terms.Exp) *Matrix {
	a, err := m.Sum(n, scale)
	if err != nil {
		panic(err)
	}
	return a
}

// Performs a substitution on all elements of a matrix.
func (m *Matrix) Substitute(b []factor.Value, s *terms.Exp) *Matrix {
	n, _ := NewMatrix(m.rows, m.cols)
	for r := 0; r < m.rows; r++ {
		for c := 0; c < m.cols; c++ {
			n.Set(r, c, m.El(r, c).Substitute(b, s))
		}
	}
	return n
}
