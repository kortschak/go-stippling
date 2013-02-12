package density

import (
	"image"
	"image/color"
)

// A common usecase is using both an SumX and a SumY map.
// The Sum type is simply an aggregrate of both, with
// some methods for added convenience.
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

func SumFrom(i image.Image, d Model) *Sum {
	r := i.Bounds()
	w, h := r.Dx(), r.Dy()
	xdv := make([]uint64, w*h)
	for y := 0; y < h; y++ {
		for x, v := 0, uint64(0); x < w; x++ {
			v += uint64(d.Convert(i.At(x+r.Min.X, y+r.Min.Y)))
			xdv[x+y*w] = v
		}
	}
	ydv := make([]uint64, w*h)
	for x := 0; x < w; x++ {
		for y, v := 0, uint64(0); y < h; y++ {
			v += uint64(d.Convert(i.At(x+r.Min.X, y+r.Min.Y)))
			ydv[x*h+y] = v
		}
	}
	return &Sum{
		X: SumX{Values: xdv, Stride: w, Rect: r},
		Y: SumY{Values: ydv, Stride: h, Rect: r},
	}
}
