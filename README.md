# Algex - a collection of Go packages for performing basic symbolic algebra

## Overview

Algex is a collection of interdependent Go packages for performing
basic algebra. The core functionality was developed to solve the
inverse kinematics of a [6-axis
robot](https://github.com/tinkerator/saxis). However, structuring it
as a set of stand alone packages provided a motivation for expanding
its capabilities.

## How to use

The documentation for each of the packages can be found on
[go.dev](https://go.dev) via this top-level search:
[zappem.net/pub/math/algex](https://pkg.go.dev/zappem.net/pub/math/algex).

The basic command line example is built as follows:
```
$ go build examples/algex.go
$ ./algex
> x:=a+b
> y:=a-b
> x*y
 a^2-b^2
> exit
exiting
```

## Other included examples

The other included example is `examples/ik.go` which is the mentioned
inverse kinematics algebra for the [saxis
robot](https://github.com/tinkerator/saxis):
```
$ go run examples/ik.go
... a lot of formulas (investigated and solved by hand) ...
```

## Features planned

- Rational polynomial factorization isn't yet implemented. Currently, we
only cancel common simple numerical factors of the denominator and the
numerator, and whole pieces of the denominator. Example:
```
$ ./algex
> (x+y)*(x-y)/(x+y)
 x-y
> 2*(x+y)*(x-y)/(x^2-y^2)
 2
> (x+y)*(x-y)/(x+y)^2
 (x^2-y^2)/(2*x*y+x^2+y^2)
> exit
exiting
```
- A LaTeX mode for rendering an expression.

## License info

The `algex` package is distributed with the same BSD 3-clause license
as that used by [golang](https://golang.org/LICENSE) itself.

## Reporting bugs

Use the [github `algex` bug
tracker](https://github.com/tinkerator/algex/issues).