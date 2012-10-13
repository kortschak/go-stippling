package density

import (
	"image/color"
)

// A Model can convert any color to a density.
type Model interface {
	Convert(c color.Color) uint16
}

// ModelFunc returns a Model that invokes f to implement the conversion.
func ModelFunc(f func(color.Color) uint16) Model {
	return &modelFunc{f}
}

type modelFunc struct {
	f func(color.Color) uint16
}

func (m *modelFunc) Convert(c color.Color) (d uint16) {
	return m.f(c)
}

// Default models for density functions. These all linearly map their
// respective channels to a density value.
var (
	AvgDensity      Model = ModelFunc(avgDensity)
	RedDensity      Model = ModelFunc(redDensity)
	GreenDensity    Model = ModelFunc(greenDensity)
	BlueDensity     Model = ModelFunc(blueDensity)
	AlphaDensity    Model = ModelFunc(alphaDensity)
	NegAvgDensity   Model = ModelFunc(negAvgDensity)
	NegRedDensity   Model = ModelFunc(negRedDensity)
	NegGreenDensity Model = ModelFunc(negGreenDensity)
	NegBlueDensity  Model = ModelFunc(negBlueDensity)
	NegAlphaDensity Model = ModelFunc(negAlphaDensity)
)

func avgDensity(c color.Color) (d uint16) {
	r, g, b, _ := c.RGBA()
	d = uint16((r + g + b) / 3)
	return
}

func redDensity(c color.Color) (d uint16) {
	r, _, _, _ := c.RGBA()
	d = uint16(r)
	return
}

func greenDensity(c color.Color) (d uint16) {
	_, g, _, _ := c.RGBA()
	d = uint16(g)
	return
}

func blueDensity(c color.Color) (d uint16) {
	_, _, b, _ := c.RGBA()
	d = uint16(b)
	return
}

func alphaDensity(c color.Color) (d uint16) {
	_, _, _, a := c.RGBA()
	d = uint16(a)
	return
}

func negAvgDensity(c color.Color) (d uint16) {
	r, g, b, _ := c.RGBA()
	d = uint16(0xFFFF - (r+g+b)/3)
	return
}

func negRedDensity(c color.Color) (d uint16) {
	r, _, _, _ := c.RGBA()
	d = uint16(0xFFFF - r)
	return
}

func negGreenDensity(c color.Color) (d uint16) {
	_, g, _, _ := c.RGBA()
	d = uint16(0xFFFF - g)
	return
}

func negBlueDensity(c color.Color) (d uint16) {
	_, _, b, _ := c.RGBA()
	d = uint16(0xFFFF - b)
	return
}

func negAlphaDensity(c color.Color) (d uint16) {
	_, _, _, a := c.RGBA()
	d = uint16(0xFFFF - a)
	return
}
