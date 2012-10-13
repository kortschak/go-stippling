package density

import (
	"image"
	"image/color"
)

// SumX is similar to Map, but the value at (x, y) is the sum of 
// all the values along the row from (0, y) up to and including
// (x, y), as converted by the density function. Note that 
// At(x, y) produces the same colour output as a regular map.
type SumX struct {
	sumMap
}

func (d *SumX) Copy(s *SumX) {
	d.sumMap.Copy(&s.sumMap)
}

func (d *SumX) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	i := d.DVOffSet(x, y)
	d.Values[i] = d.ValueAt(x-1, y) + uint64(v)
}

func (d *SumX) At(x, y int) (v color.Color) {
	v = color.Gray16{uint16(d.ValueAt(x, y) - d.ValueAt(x-1, y))}
	return
}

func newSumX(r image.Rectangle) (sx *SumX) {
	w, h := r.Dx(), r.Dy()
	if w > 0 && h > 0 {
		dv := make([]uint64, w*h)
		sx = &SumX{sumMap{Values: dv, Stride: w, Rect: r}}
	}
	return
}

func SumXFrom(i image.Image, d Model) *SumX {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	sx := SumX{sumMap{Values: dv, Stride: w, Rect: r}}

	// TODO:
	// This can be optimised. A lot. Split the inner loop into two for loops:
	// One for the leftmost column, one for the other values. This removes
	// the need for the bounds checking done in InitSet.
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			sx.Set(x, y, d.Convert(i.At(x, y)))
		}
	}
	return &sx
}
