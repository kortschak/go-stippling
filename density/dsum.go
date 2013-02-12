package density

import (
	"image"
	"image/color"
)

// DSum is similar to Map, but the value at (x, y) is the sum of all
// the values in the rectangle (0,0) - (x, y), as converted by the
// density function (a Double Sum). Note that At(x, y) produces the
// the same colour output as a regular map.
type DSum struct {
	// Values holds the map's density values. The value at (x, y)
	// starts at Values[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*1].
	Values []uint64
	// Stride is the Values' stride between
	// vertically adjacent pixels.
	Stride int
	// Rect is the Map's bounds.
	Rect image.Rectangle
}

func (d *DSum) ColorModel() color.Model {
	return color.Gray16Model
}

func (d *DSum) Copy(s *DSum) {
	d.Values = make([]uint64, len(s.Values), cap(s.Values))
	copy(d.Values, s.Values)
	d.Stride = s.Stride
	d.Rect = s.Rect
}

func (d *DSum) DVOffSet(x, y int) int {
	return (y-d.Rect.Min.Y)*d.Stride + (x - d.Rect.Min.X)
}

func (d *DSum) Bounds() image.Rectangle { return d.Rect }

func (d *DSum) At(x, y int) (v color.Color) {
	// v = color.Gray16{d.ValueAt(x, y) - d.ValueAt(x, y-1) - d.ValueAt(x-1, y) + d.ValueAt(x-1, y-1)}
	if (image.Point{x, y}.In(d.Rect)) {
		i := d.DVOffSet(x, y)

		if x-d.Rect.Min.X > 0 {
			if y-d.Rect.Min.Y > 0 {
				v = color.Gray16{uint16(d.Values[i-(d.Stride+1)] -
					d.Values[i-d.Stride] -
					d.Values[i-1] +
					d.Values[i]),
				}
			} else {
				v = color.Gray16{uint16(d.Values[i] - d.Values[i-1])}
			}
		} else if y-d.Rect.Min.Y > 0 {
			v = color.Gray16{uint16(d.Values[i] - d.Values[i-d.Stride])}
		} else {
			v = color.Gray16{uint16(d.Values[0])}
		}
	}
	return
}

func (d *DSum) ValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(d.Rect)) {
		i := d.DVOffSet(x, y)
		v = d.Values[i]
	}
	return
}

// Note that when you set a value at (x,y), the entire
// area covered from (x,y) to the bottom right has to
// be updated. In other words: very slow operation
func (d *DSum) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	// First, convert to the delta of the value at (x,y)
	var dv uint64
	dv = uint64(v) - d.ValueAt(x, y) - d.ValueAt(x-1, y-1) + d.ValueAt(x-1, y) + d.ValueAt(x, y-1)

	//now apply to all affected part of the DSum
	for j := y; j < d.Rect.Max.Y; j++ {
		for i := x; i < d.Rect.Max.X; i++ {
			d.Values[d.DVOffSet(i, j)] += dv
		}
	}
}

// Sums the rectangle r from r.Min up to but not including r.Max
func (d *DSum) AreaSum(r image.Rectangle) uint64 {
	r = r.Intersect(d.Rect)
	return d.ValueAt(r.Max.X-1, r.Max.Y-1) +
		d.ValueAt(r.Min.X-1, r.Min.Y-1) -
		d.ValueAt(r.Min.X-1, r.Max.Y-1) -
		d.ValueAt(r.Max.X-1, r.Min.Y-1)
}

func NewDSum(r image.Rectangle) DSum {
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	return DSum{Values: dv, Stride: w, Rect: r}
}

func DSumFrom(i image.Image, d Model) *DSum {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)

	for x, vx := 0, uint64(0); x < w; x++ {
		vx += uint64(d.Convert(i.At(x+r.Min.X, r.Min.Y)))
		dv[x] = vx
	}

	for y := 1; y < h; y++ {
		for x, vx := 0, uint64(0); x < w; x++ {
			vx += uint64(d.Convert((i.At(x+r.Min.X, y+r.Min.Y))))
			dv[x+y*w] = vx + dv[x+(y-1)*w]
		}
	}

	return &DSum{Values: dv, Stride: w, Rect: r}
}
