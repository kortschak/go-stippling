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
	r := d.Bounds()

	// First, convert to the delta of the value at (x,y)
	i := d.DVOffSet(x, y)
	var dv uint64
	if x == r.Min.X {
		dv = uint64(v) - d.Values[i]
	} else {
		dv = uint64(v) - d.Values[i] + d.Values[i-1]
	}

	// Now create a delta slice from x to r.Max.X
	// while updating the current line
	ds := make([]uint64, r.Max.X-x)
	if y == r.Min.Y {
		for j := 0; j < len(ds); j++ {
			d.Values[i+j] += dv
			ds[j] = d.Values[i+j]
		}
	} else {
		for j := 0; j < len(ds); j++ {
			d.Values[i+j] += dv
			ds[j] = d.Values[i+j] - d.Values[i+j-d.Stride]
		}
	}

	// Now update all lines below
	for y++; y < r.Max.Y; y++ {
		i += d.Stride
		for j := 0; j < len(ds); j++ {
			d.Values[i+j] += ds[j]
		}
	}
}

func (d *DSum) AreaSum(x0, y0, x1, y1 int) uint64 {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}

	return d.ValueAt(x1, y1) + d.ValueAt(x0, y0) - d.ValueAt(x0, y1) - d.ValueAt(x1, y0)
}

func NewDSum(r image.Rectangle) DSum {
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	return DSum{Values: dv, Stride: w, Rect: r}
}

func DSumFrom(i image.Image, d Model) DSum {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)

	dv[0] = uint64(d.Convert(i.At(r.Min.X, r.Min.Y)))

	for j, x := 0, r.Min.X; x < r.Max.X; x++ {
		j++
		dv[j] = dv[j-1] + uint64(d.Convert(i.At(x, r.Min.Y)))
	}

	for j, y := 0, r.Min.Y+1; y < r.Max.Y; y++ {
		j += w
		dv[j] = dv[j-w] + uint64(d.Convert(i.At(r.Min.X, y)))
	}

	for j, y := 0, r.Min.Y+1; y < r.Max.Y; y++ {
		j += w
		for x := r.Min.X + 1; x < r.Max.X; x++ {
			j++
			dv[j] = dv[j-w] + dv[j-1] + uint64(d.Convert(i.At(x, y)))
		}
	}

	return DSum{Values: dv, Stride: w, Rect: r}
}
