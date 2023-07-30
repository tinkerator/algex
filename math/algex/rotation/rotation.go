// Package rotation generates matrices for 3D rotations.
//
// This package prefixes angles with 's', 'c' or 't' for sine, cosine
// and tangent respectively.
package rotation

import (
	"zappem.net/pub/math/algex/factor"
	"zappem.net/pub/math/algex/matrix"
	"zappem.net/pub/math/algex/terms"
)

var one = terms.NewExp([]factor.Value{factor.D(1, 1)})

// A matrix for rotating anticlockwise around the X-axis.
func RX(theta string) *matrix.Matrix {
	m, _ := matrix.NewMatrix(3, 3)
	c, _ := terms.ParseExp("c" + theta)
	s, _ := terms.ParseExp("s" + theta)
	mS, _ := terms.ParseExp("-s" + theta)

	m.Set(0, 0, one)
	m.Set(1, 1, c)
	m.Set(2, 2, c)

	m.Set(1, 2, mS)
	m.Set(2, 1, s)
	return m
}

// A matrix for rotating anticlockwise around the Y-axis.
func RY(theta string) *matrix.Matrix {
	m, _ := matrix.NewMatrix(3, 3)
	c, _ := terms.ParseExp("c" + theta)
	s, _ := terms.ParseExp("s" + theta)
	mS, _ := terms.ParseExp("-s" + theta)

	m.Set(0, 0, c)
	m.Set(1, 1, one)
	m.Set(2, 2, c)

	m.Set(0, 2, s)
	m.Set(2, 0, mS)

	return m
}

// A matrix for rotating anticlockwise around the Z-axis.
func RZ(theta string) *matrix.Matrix {
	m, _ := matrix.NewMatrix(3, 3)
	c, _ := terms.ParseExp("c" + theta)
	s, _ := terms.ParseExp("s" + theta)
	mS, _ := terms.ParseExp("-s" + theta)

	m.Set(0, 0, c)
	m.Set(1, 1, c)
	m.Set(2, 2, one)

	m.Set(0, 1, mS)
	m.Set(1, 0, s)

	return m
}
