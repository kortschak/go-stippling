// Package density implements functions to map pixels in an image 
// to associated density values, and store them in a Map (not to
// be confused with the built-in type). It is mostly an adaptation
// of the code found in the image package, and still uses Point 
// and Rect from that package.
//
// Density values of a pixel as defined by the density model are 
// stored as uint16 values. Maps implement the Image interface, as
// Gray16 images (obviously).
//
// Although written for making weighted voronoi maps based on 
// images, it probably can be used more widely than that.
//
// SumX, SumY and DSum are special density Maps that store summed 
// density values values over the X-axis, the Y-axis, and both X 
// and Y axes respectively. When summing over a large area of the
// same Map repeatedly this can be faster, as it can greatly
// reduce the number of memory lookups. Take note that this also
// greatly depends on things like branch prediction.
package density

import (
	"image"
	"image/color"
)

// Map is a finite rectangular grid of density values, usually 
// converted from the colors of an image. 
type Map struct {
	// Values holds the map's density values. The value at (x, y) 
	// starts at Values[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*1].
	Values []uint16
	// Stride is the Values' stride between 
	// vertically adjacent pixels.
	Stride int
	// Rect is the Map's bounds.
	Rect image.Rectangle
	// Total mass, weighed x and weighed y. Essentially a cache to
	// speed up a number of calculations.
	mass, wx, wy uint64
}

func (d *Map) Copy(s *Map) {
	d.Values = make([]uint16, len(s.Values), cap(s.Values))
	copy(d.Values, s.Values)
	d.Stride = s.Stride
	d.Rect = s.Rect
	d.mass = s.mass
	d.wx = s.wx
	d.wy = s.wy
}

// The density map has Gray16 as its colormodel
func (d *Map) ColorModel() color.Model {
	return color.Gray16Model
}

// DVOffset returns the index that corresponds to Values 
// at (x, y).
func (d *Map) DVOffSet(x, y int) int {
	return (y-d.Rect.Min.Y)*d.Stride + (x - d.Rect.Min.X)
}

func (d *Map) Set(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	i := d.DVOffSet(x, y)

	// We update mass, wx and wy by removing the
	// old value first, then adding the new value.
	dv := uint64(d.Values[i])

	d.mass -= dv
	d.wx -= dv * uint64(x-d.Rect.Min.X)
	d.wy -= dv * uint64(y-d.Rect.Min.Y)

	d.Values[i] = v
	dv = uint64(v)

	d.mass += dv
	d.wx += dv * uint64(x-d.Rect.Min.X)
	d.wy += dv * uint64(y-d.Rect.Min.Y)

}

// InitSet(x, y) is almost identical to Set(x, y), but assumes the
// previous value at (x,y) was 0. Use it to speed up constructors.
func (d *Map) InitSet(x, y int, v uint16) {
	if !(image.Point{x, y}.In(d.Rect)) {
		return
	}
	i := d.DVOffSet(x, y)

	d.Values[i] = v

	// We update mass, wx and wy. Since the original values
	// were zero, we can immediately add the new value.
	dv := uint64(v)

	d.mass += dv
	d.wx += dv * uint64(x-d.Rect.Min.X)
	d.wy += dv * uint64(y-d.Rect.Min.Y)
}

func (d *Map) Bounds() image.Rectangle { return d.Rect }

// At(x, y) returns the density value at point (x,y).
// If (x,y) is out of bounds, it returns a density of zero.
func (d *Map) At(x, y int) (v color.Color) {
	if (image.Point{x, y}.In(d.Rect)) {
		i := d.DVOffSet(x, y)
		v = color.Gray16{d.Values[i]}
	}
	return
}

// ValueAt(x, y) returns the density value at point (x,y), but as a uint16
// instead of a color.Color interface. If (x,y) is out of bounds, it 
// returns a density of zero.
func (d *Map) ValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(d.Rect)) {
		i := d.DVOffSet(x, y)
		v = uint64(d.Values[i])
	}
	return
}

// CM returns the centre of mass of the Map.
func (d *Map) CM() (x, y float64) {
	x = float64(d.Rect.Min.X) + (float64(d.wx) / float64(d.mass))
	y = float64(d.Rect.Min.Y) + (float64(d.wy) / float64(d.mass))
	return
}

// Mass returns the mass of the density map - in other words:
// it's density integrated over it's surface.
func (d *Map) Mass() uint64 {
	return d.mass
}

// WX returns the weighted X of the density map. Note that it is not
// bounds-corrected (that is: it takes the top-left corner of the 
// map to be at point (0,0) instead of (Rect.Min.X, Rect.Min.Y))
func (d *Map) WX() uint64 {
	return d.wx
}

// WY returns the weighted Y of the density map. Note that it is not
// bounds-corrected (that is: it takes the top-left corner of the 
// map to be at point (0,0) instead of (Rect.Min.X, Rect.Min.Y))
func (d *Map) WY() uint64 {
	return d.wy
}

// AvgDens returns the average density of the Map.
func (d *Map) AvgDens() (v float64) {
	return float64(d.mass) / float64(d.Rect.Dx()*d.Rect.Dy())
}

// NewMap returns an empty map of the given dimensions.
func NewMap(r image.Rectangle) (d *Map) {
	w, h := r.Dx(), r.Dy()
	if w > 0 && h > 0 {
		dv := make([]uint16, w*h)
		d = &Map{Values: dv, Stride: w, Rect: r}
	}
	return
}

// Determines the density values of image.Image according to the density 
// model it is given, and returns the results as a new Map.
func MapFrom(i image.Image, d Model) *Map {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint16, w*h)
	dm := Map{Values: dv, Stride: w, Rect: r}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dm.InitSet(x, y, d.Convert(i.At(x, y)))
		}
	}
	return &dm
}

// SubMap returns a Map representing the portion of the Map d visible 
// through r. The returned map shares values with the original map.
func (d *Map) SubMap(r image.Rectangle) (s *Map) {
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
	ym := uint64(r.Dy())
	xm := uint64(r.Dx())
	for y := uint64(0); y < ym; y++ {
		for x := uint64(0); x < xm; x++ {
			m := uint64(sv[y*uint64(d.Stride)+x])
			sm += m
			swx += m * x
			swy += m * y
		}
	}

	s = &Map{
		Values: sv,
		Stride: d.Stride,
		Rect:   r,
		mass:   sm,
		wx:     swx,
		wy:     swy,
	}
	return
}
