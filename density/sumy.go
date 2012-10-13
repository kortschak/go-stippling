package density

import (
	"image"
	"image/color"
)

// SumY is similar to Map, but the value at (x, y) is the sum of all 
// the values along the row from (0, y) up to and including (x, y),
// as converted by the density function. Note that At(x, y) produces
// the same colour output as a regular map.
//
// For cache purposes, the data is internally stored as a transposed
// array, that is: (x, y) = Values[x*height + y]. This is mostly done
// by simply putting a few wrapper methods around an anonymous SumX 
// struct (and hoping the compiler inlines these methods).
type SumY struct {
	SumX
}

func (d *SumY) Copy(s *SumY) {
	d.SumX.Copy(&s.SumX)
}

func (d *SumY) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}

	// We update mass, wx and wy by removing the
	// old value first, then adding the new value.
	i := d.DVOffSet(x, y)
	d.Values[i] = d.ValueAt(x, y-1) + uint64(v)
}

func (d *SumY) DVOffSet(x, y int) int {
	return (y - d.Rect.Min.Y) + (x-d.Rect.Min.X)*d.Stride
}

func (d *SumY) At(x, y int) (v color.Color) {
	v = color.Gray16{uint16(d.ValueAt(x, y) - d.ValueAt(x, y-1))}
	return
}

func (d *SumY) ValueAt(x, y int) (v uint64) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	i := d.DVOffSet(x, y)
	v = d.Values[i]
	return
}

func newSumY(r image.Rectangle) (sy *SumY) {
	w, h := r.Dx(), r.Dy()
	if w > 0 && h > 0 {
		dv := make([]uint64, w*h)
		sy = &SumY{SumX{sumMap{Values: dv, Stride: h, Rect: r}}}
	}
	return
}

func SumYFrom(i image.Image, d Model) *SumY {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	sy := SumY{SumX{sumMap{Values: dv, Stride: h, Rect: r}}}

	// TODO:
	// This can be optimised. A lot. Split the inner loop into two for loops:
	// One for the topmost row, one for the other values. This removes the
	// need for the bounds checking done in InitSet.
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			sy.Set(x, y, d.Convert(i.At(x, y)))
		}
	}
	return &sy
}
