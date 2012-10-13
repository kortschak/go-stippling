package density

import (
	"image"
	"image/color"
)

// sumMap is used internally for all summed maps, since the main
// difference between these types of maps is the way they are
// constructed. The only internal difference between sumMap and a
// standard density map is that it uses uint64 instead of uint16,
// as it has to sum many density values.
type sumMap struct {
	// Values holds the map's density values. The value at (x, y) 
	// starts at Values[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*1].
	Values []uint64
	// Stride is the Values' stride between 
	// vertically adjacent pixels.
	Stride int
	// Rect is the Map's bounds.
	Rect image.Rectangle
}

func (d *sumMap) ColorModel() color.Model {
	return color.Gray16Model
}

func (d *sumMap) Copy(s *sumMap) {
	d.Values = make([]uint64, len(s.Values), cap(s.Values))
	copy(d.Values, s.Values)
	d.Stride = s.Stride
	d.Rect = s.Rect
}

func (d *sumMap) DVOffSet(x, y int) int {
	return (y-d.Rect.Min.Y)*d.Stride + (x - d.Rect.Min.X)
}

func (d *sumMap) Bounds() image.Rectangle { return d.Rect }

func (d *sumMap) ValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(d.Rect)) {
		i := d.DVOffSet(x, y)
		v = d.Values[i]
	}
	return
}
