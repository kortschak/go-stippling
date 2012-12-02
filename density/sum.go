package density

import (
	"image"
	"image/color"
)

// A common usecase is using both an SumX and a SumY map.
// The Sum type is simply an aggregrate of both, with
// some methods for added convenience. Use together with
// SumMask to calculate weighed x, y and mass over an
// area fast. 
type Sum struct {
	X SumX
	Y SumY
}

func (d *Sum) Copy(s *Sum) {
	d.X.Copy(&s.X)
	d.Y.Copy(&s.Y)
}

func (d *Sum) Bounds() image.Rectangle {
	return d.X.Bounds()
}

func (d *Sum) At(x, y int) (v color.Color) {
	return d.X.At(x, y)
}

func (d *Sum) ColorModel() color.Model {
	return color.Gray16Model
}

func SumFrom(i image.Image, d Model) Sum {
	return Sum{SumXFrom(i, d), SumYFrom(i, d)}
}
