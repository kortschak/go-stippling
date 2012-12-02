package density

import (
	"image"
	"image/color"
)

// SumY is similar to Map, but the value retrieved by ValueAt(x, y) is
// the sum of all the values along the column from (x, 0) up to and
// including (x, y), as converted by the density function.
// At(x, y) produces the same colour output as a regular map.
//
// For cache purposes, the data is internally stored as a transposed
// array, that is: (x, y) = Values[x*height + y].
type SumY struct {
	// Values holds the map's density values. The value at (x, y)
	// starts at Values[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*1].
	Values []uint64
	// Stride is the Values' stride between
	// vertically adjacent pixels.
	Stride int
	// Rect is the Map's bounds.
	Rect image.Rectangle
}

func (d *SumY) ColorModel() color.Model {
	return color.Gray16Model
}

func (d *SumY) Copy(s *SumY) {
	d.Values = make([]uint64, len(s.Values), cap(s.Values))
	copy(d.Values, s.Values)
	d.Stride = s.Stride
	d.Rect = s.Rect
}

func (d *SumY) DVOffSet(x, y int) int {
	return (y - d.Rect.Min.Y) + (x-d.Rect.Min.X)*d.Stride
}

func (d *SumY) Bounds() image.Rectangle { return d.Rect }

func (d *SumY) At(x, y int) color.Color {
	return color.Gray16{uint16(d.ValueAt(x, y) - d.ValueAt(x, y-1))}
}

func (d *SumY) ValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(d.Rect)) {
		i := d.DVOffSet(x, y)
		v = d.Values[i]
	}
	return
}

func (d *SumY) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	// First, convert to the delta of the value at (x,y)
	i := d.DVOffSet(x, y)
	var dv uint64
	if y == d.Bounds().Min.Y {
		dv = uint64(v) - d.Values[i]
	} else {
		dv = uint64(v) - d.Values[i] + d.Values[i-1]
	}

	// Now, update the column
	for mi := i + d.Bounds().Max.Y - x; i < mi; i++ {
		d.Values[i] += dv
	}
}

func NewSumY(r image.Rectangle) SumY {
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	return SumY{Values: dv, Stride: h, Rect: r}
}

func SumYFrom(i image.Image, d Model) SumY {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	for j, x := 0, r.Min.X; x < r.Max.X; x++ {
		dv[j] = uint64(d.Convert(i.At(x, r.Min.Y)))
		j += h
	}
	for j, x := 0, r.Min.X; x < r.Max.X; x++ {
		j++ // skip the first line
		for y := r.Min.Y + 1; y < r.Max.Y; y++ {
			dv[j] = dv[j-1] + uint64(d.Convert(i.At(x, y)))
			j++
		}
	}

	return SumY{Values: dv, Stride: w, Rect: r}

}
