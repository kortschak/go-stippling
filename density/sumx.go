package density

import (
	"image"
	"image/color"
)

// SumX is similar to Map, but the value retrieved by ValueAt(x, y) is
// the sum of all the values along the row from (0, y) up to and
// including (x, y), as converted by the density function.
// At(x, y) produces the same colour output as a regular map.
type SumX struct {
	// Values holds the map's density values. The value at (x, y)
	// starts at Values[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*1].
	Values []uint64
	Stride int
	// Rect is the Map's bounds.
	Rect image.Rectangle
}

func (d *SumX) ColorModel() color.Model {
	return color.Gray16Model
}

func (d *SumX) Copy(s *SumX) {
	d.Values = make([]uint64, len(s.Values), cap(s.Values))
	copy(d.Values, s.Values)
	d.Stride = s.Stride
	d.Rect = s.Rect
}

// Does not do bounds checking!
func (d *SumX) DVOffSet(x, y int) int {
	return (y-d.Rect.Min.Y)*d.Stride + (x - d.Rect.Min.X)
}

func (d *SumX) Bounds() image.Rectangle { return d.Rect }

func (d *SumX) At(x, y int) color.Color {
	return color.Gray16{uint16(d.ValueAt(x, y) - d.ValueAt(x-1, y))}
}

func (d *SumX) ValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(d.Rect)) {
		v = d.Values[(y-d.Rect.Min.Y)*d.Stride+(x-d.Rect.Min.X)]
	}
	return
}

func (d *SumX) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	// First, convert to the delta of the value at (x,y)
	i := d.DVOffSet(x, y)
	var dv = uint64(v)
	if x == d.Rect.Min.X {
		dv -= d.Values[i]
	} else {
		dv -= d.Values[i] - d.Values[i-1]
	}

	// Now, update the line
	for mi := i + d.Rect.Max.X - x; i < mi; i++ {
		d.Values[i] += dv
	}
}

func NewSumX(r image.Rectangle) *SumX {
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	return &SumX{Values: dv, Stride: w, Rect: r}
}

func SumXFrom(i image.Image, d Model) *SumX {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	for y := 0; y < h; y++ {
		for x, v := 0, uint64(0); x < w; x++ {
			v += uint64(d.Convert(i.At(x+r.Min.X, y+r.Min.Y)))
			dv[x+y*w] = v
		}
	}
	return &SumX{Values: dv, Stride: w, Rect: r}
}
