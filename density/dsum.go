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
	sumMap
}

func (d *DSum) Copy(s *DSum) {
	d.sumMap.Copy(&s.sumMap)
}
func (d *DSum) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	i := d.DVOffSet(x, y)
	d.Values[i] = uint64(v) - d.ValueAt(x-1, y-1) + d.ValueAt(x, y-1) + d.ValueAt(x-1, y)
}

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

func (d *DSum) AreaSum(x0, y0, x1, y1 int) (v uint64) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}

	v = d.ValueAt(x1, y1) + d.ValueAt(x0, y0) - d.ValueAt(x0, y1) - d.ValueAt(x1, y0)
	return
}

func newDSum(r image.Rectangle) (d *DSum) {
	w, h := r.Dx(), r.Dy()
	if w > 0 && h > 0 {
		dv := make([]uint64, w*h)
		d = &DSum{sumMap{Values: dv, Stride: w, Rect: r}}
	}
	return
}

func DSumFrom(i image.Image, d Model) *DSum {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)
	ds := DSum{sumMap{Values: dv, Stride: w, Rect: r}}

	//TODO: OPTIMISE. See SumX/SumY
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			ds.Set(x, y, d.Convert(i.At(x, y)))
		}
	}
	return &ds
}

/*
func (d *DSum) SubDSum(r image.Rectangle) (s *DSum) {
	r = r.Intersect(d.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Values[i:] expression below can panic.
	if r.Empty() {
		return
	}

	i := d.DVOffSet(r.Min.X, r.Min.Y)

	sv := d.Values[i:]
	// Recalculate the mass, weighed x and weighed y
	var sm, swx, swy uint64
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			m := d.ValueAt(x, y) - d.ValueAt(x, y-1)
			sm += m
			swx += m * x
			swy += m * y
		}
	}

	s = &SumY{
		SumX{
			sumMap{
				Values: sv,
				Stride: d.Stride,
				Rect:   r,
				mass:   sm,
				wx:     swx,
				wy:     swy,
			},
		},
	}
	return
}
*/
