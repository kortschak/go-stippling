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

func (d *DSum) NegValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(d.Rect)) {
		i := d.DVOffSet(x, y)
		v = uint64(x+1-d.Rect.Min.X)*uint64(y+1-d.Rect.Min.Y)*0xFFFF - d.Values[i]
	}
	return
}

// Given a Rectangle, finds x closest to line dividing
// the mass of the area bound by these coordinates in half.
func (ds *DSum) FindCx(r image.Rectangle) int {
	r = ds.Rect.Intersect(r)
	xmin := r.Min.X
	xmax := r.Max.X
	r = r.Sub(image.Point{1, 1})
	x := (xmax + xmin + 1) / 2
	xmaxymax := ds.ValueAt(r.Max.X, r.Max.Y)
	xminymin := ds.ValueAt(r.Min.X, r.Min.Y)
	xminymax := ds.ValueAt(r.Min.X, r.Max.Y)
	xmaxymin := ds.ValueAt(r.Max.X, r.Min.Y)
	for {
		// The centre of mass is probably not a round number,
		// so we aim to iterate only to the margin of 1 pixel
		if xmax-xmin > 1 {
			cxymin := ds.ValueAt(x, r.Min.Y)
			cxymax := ds.ValueAt(x, r.Max.Y)
			lmass := cxymax - cxymin - xminymax + xminymin
			rmass := xmaxymax - cxymax - xmaxymin + cxymin
			if lmass < rmass {
				xmin = x
				x = (x + xmax + 1) / 2
			} else {
				xmax = x
				x = (x + xmin + 1) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Since both are rounded down, that means the biggest of the two.
			cxymin := ds.ValueAt(xmin, r.Min.Y)
			cxymax := ds.ValueAt(xmin, r.Max.Y)
			lmass := cxymax - cxymin - xminymax + xminymin
			cxymin = ds.ValueAt(xmax, r.Min.Y)
			cxymax = ds.ValueAt(xmax, r.Max.Y)
			rmass := xmaxymax - cxymax - xmaxymin + cxymin
			if lmass > rmass {
				x = xmin
			} else {
				x = xmax
			}
			break
		}
	}
	return x
}

// Given a Rectangle, finds y closest to line dividing
// the mass of the area bound by these coordinates in half.
func (ds *DSum) FindCy(r image.Rectangle) int {
	r = ds.Rect.Intersect(r)
	ymin := r.Min.Y
	ymax := r.Max.Y
	r = r.Sub(image.Point{1, 1})
	y := (ymax + ymin + 1) / 2
	xmaxymax := ds.ValueAt(r.Max.X, r.Max.Y)
	xminymin := ds.ValueAt(r.Min.X, r.Min.Y)
	xminymax := ds.ValueAt(r.Min.X, r.Max.Y)
	xmaxymin := ds.ValueAt(r.Max.X, r.Min.Y)
	for {
		// The centre of mass is probably not a round number,
		// so we aim to iterate only to the margin of 1 pixel
		if ymax-ymin > 1 {
			xmincy := ds.ValueAt(r.Min.X, y)
			xmaxcy := ds.ValueAt(r.Max.X, y)
			tmass := xmaxcy - xmincy - xmaxymin + xminymin
			dmass := xmaxymax - xminymax - xmaxcy + xmincy
			if tmass < dmass {
				ymin = y
				y = (y + ymax + 1) / 2
			} else {
				ymax = y
				y = (y + ymin + 1) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Since both are rounded down, that means the biggest of the two.
			xmincy := ds.ValueAt(r.Min.X, ymin)
			xmaxcy := ds.ValueAt(r.Max.X, ymin)
			tmass := xmaxcy - xmincy - xmaxymin + xminymin
			xmincy = ds.ValueAt(r.Min.X, ymax)
			xmaxcy = ds.ValueAt(r.Max.X, ymax)
			dmass := xmaxymax - xminymax - xmaxcy + xmincy
			if tmass > dmass {
				y = ymin
			} else {
				y = ymax
			}
			break
		}
	}
	return y
}

// Given a Rectangle, finds x closest to line dividing
// the negative mass of the area bound by these coordinates in half.
func (ds *DSum) FindNegCx(r image.Rectangle) int {
	r = ds.Rect.Intersect(r)
	xmin := r.Min.X
	xmax := r.Max.X
	r = r.Sub(image.Point{1, 1})
	x := (xmax + xmin + 1) / 2
	xmaxymax := ds.NegValueAt(r.Max.X, r.Max.Y)
	xminymin := ds.NegValueAt(r.Min.X, r.Min.Y)
	xminymax := ds.NegValueAt(r.Min.X, r.Max.Y)
	xmaxymin := ds.NegValueAt(r.Max.X, r.Min.Y)
	for {
		// The centre of mass is probably not a round number,
		// so we aim to iterate only to the margin of 1 pixel
		if xmax-xmin > 1 {
			cxymin := ds.NegValueAt(x, r.Min.Y)
			cxymax := ds.NegValueAt(x, r.Max.Y)
			lmass := cxymax - cxymin - xminymax + xminymin
			rmass := xmaxymax - cxymax - xmaxymin + cxymin
			if lmass < rmass {
				xmin = x
				x = (x + xmax + 1) / 2
			} else {
				xmax = x
				x = (x + xmin + 1) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Since both are rounded down, that means the biggest of the two.
			cxymin := ds.NegValueAt(xmin, r.Min.Y)
			cxymax := ds.NegValueAt(xmin, r.Max.Y)
			lmass := cxymax - cxymin - xminymax + xminymin
			cxymin = ds.NegValueAt(xmax, r.Min.Y)
			cxymax = ds.NegValueAt(xmax, r.Max.Y)
			rmass := xmaxymax - cxymax - xmaxymin + cxymin
			if lmass > rmass {
				x = xmin
			} else {
				x = xmax
			}
			break
		}
	}
	return x
}

// Given a Rectangle, finds y closest to line dividing
// the negative mass of the area bound by these coordinates in half.
func (ds *DSum) FindNegCy(r image.Rectangle) int {
	r = ds.Rect.Intersect(r)
	ymin := r.Min.Y
	ymax := r.Max.Y
	r = r.Sub(image.Point{1, 1})
	y := (ymax + ymin + 1) / 2
	xmaxymax := ds.NegValueAt(r.Max.X, r.Max.Y)
	xminymin := ds.NegValueAt(r.Min.X, r.Min.Y)
	xminymax := ds.NegValueAt(r.Min.X, r.Max.Y)
	xmaxymin := ds.NegValueAt(r.Max.X, r.Min.Y)
	for {
		// The centre of mass is probably not a round number,
		// so we aim to iterate only to the margin of 1 pixel
		if ymax-ymin > 1 {
			xmincy := ds.NegValueAt(r.Min.X, y)
			xmaxcy := ds.NegValueAt(r.Max.X, y)
			tmass := xmaxcy - xmincy - xmaxymin + xminymin
			dmass := xmaxymax - xminymax - xmaxcy + xmincy
			if tmass < dmass {
				ymin = y
				y = (y + ymax + 1) / 2
			} else {
				ymax = y
				y = (y + ymin + 1) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Since both are rounded down, that means the biggest of the two.
			xmincy := ds.NegValueAt(r.Min.X, ymin)
			xmaxcy := ds.NegValueAt(r.Max.X, ymin)
			tmass := xmaxcy - xmincy - xmaxymin + xminymin
			xmincy = ds.NegValueAt(r.Min.X, ymax)
			xmaxcy = ds.NegValueAt(r.Max.X, ymax)
			dmass := xmaxymax - xminymax - xmaxcy + xmincy
			if tmass > dmass {
				y = ymin
			} else {
				y = ymax
			}
			break
		}
	}
	return y
}

// Note that when you set a value at (x,y), the entire
// area covered from (x,y) to the bottom right has to
// be updated. In other words: very slow operation.
func (d *DSum) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	// First, convert to the delta of the value at (x,y)
	// Did I mention I love Go's rules for rollover?
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

func DSumFrom(i *image.Image, d Model) *DSum {
	r := (*i).Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h)

	for x, vx := 0, uint64(0); x < w; x++ {
		vx += uint64(d.Convert((*i).At(x+r.Min.X, r.Min.Y)))
		dv[x] = vx
	}

	for y := 1; y < h; y++ {
		for x, vx := 0, uint64(0); x < w; x++ {
			vx += uint64(d.Convert(((*i).At(x+r.Min.X, y+r.Min.Y))))
			dv[x+y*w] = vx + dv[x+(y-1)*w]
		}
	}

	return &DSum{Values: dv, Stride: w, Rect: r}
}
