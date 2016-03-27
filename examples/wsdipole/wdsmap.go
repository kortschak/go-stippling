// TODO - it's being built at the moment
// wdipsmap will be almost equivalent to wdipmap, except for one
// crucial thing: it uses SumMaps and SumMasks, greatly
// reducing memory footprint and speeding up computation.
package main

import (
	"github.com/kortschak/go-stippling/density"
	"image"
	"math"
)

type dipole struct {
	//North and South
	N, S *density.Sum
	Mask *density.SumMask
}

func (d *dipole) Bounds() image.Rectangle {
	return d.Mask.X.Bounds()
}

// This function returns two dipoles that divide the orignal dipole
// along the weighted dividing line between the north and south poles.
func (dp *dipole) Split() (dn, ds *dipole) {

	dp.Mask.ApplyTo(N)
	xn, yn := dp.Mask.CM()
	mn := dp.Mask.Mass()

	dp.Mask.ApplyTo(S)
	xs, ys := dp.Mask.CM()
	ms := dp.Mask.Mass()

	xc := (xn*float64(ms) + xs*float64(mn)) / float64(mn+ms)
	yc := (yn*float64(ms) + ys*float64(mn)) / float64(mn+ms)

	var dx, dy float64
	if yn > ys {
		dx = (xn - xs) / (yn - ys)
		dy := 1 / dx
	} else if ys > yn {
		dx = (xs - xn) / (yn - ys)
		dy := 1 / dx
	}

	steep := math.Abs(xn-xs) < math.Abs(yn-ys)
	r := dp.Bounds()

}

func endpoints(r image.Rectangle, dx, dy, xc, yc float64, steep bool) (p0, p1 struct{ x, y float64 }) {

}

// Firefox Score: 6674
