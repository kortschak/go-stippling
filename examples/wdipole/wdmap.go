package main

// wdipmap is almost equivalent to dipmap, except for one
// crucial thing: it makes weighted voronoi cells instead
// of regular ones.

import (
	"github.com/kortschak/go-stippling/density"
	"image"
	"image/color"
	"math"
)

type cell struct {
	Source, Mask, Result *density.Map
}

func (c *cell) changedMaps() {
	c.Result = c.Source.Intersect(c.Mask)
}

func (c *cell) ValueAt(x, y int) uint64 {
	return (c.Result.Mass() * c.Mask.ValueAt(x, y)) / c.Mask.Mass()
}

func (c *cell) Bounds() (r image.Rectangle) {
	return c.Result.Bounds()
}

func (c *cell) Mass() (m uint64) {
	return c.Result.Mass()
}

type dipole struct {
	//North and South
	N, S cell
}

func newDipole(n, s, mask *density.Map) (d *dipole) {
	d = &dipole{
		N: cell{Source: n, Mask: mask},
		S: cell{Source: s, Mask: mask},
	}
	d.N.changedMaps()
	d.S.changedMaps()
	return
}

func (d *dipole) setMask(mask *density.Map) {
	d.N.Mask = mask
	d.S.Mask = mask
	d.N.changedMaps()
	d.S.changedMaps()
}

type WDMap struct {
	dipoles []dipole
	render  []uint16
	rect    image.Rectangle
}

func (wd *WDMap) Copy(a *WDMap) {
	copy(wd.dipoles, a.dipoles)
	copy(wd.render, a.render)
	wd.rect = a.rect
}

func (wd *WDMap) ColorModel() color.Model {
	return color.Gray16Model
}

func (wd *WDMap) Bounds() image.Rectangle {
	return wd.rect
}

//Render dipoles using n goroutines
func (wd *WDMap) Render(n int) {
	if n <= 0 {
		n = 1
	}
	if len(wd.dipoles) < n {
		n = len(wd.dipoles)
	}

	nrender := make([][]uint64, n)

	for i := 0; i < n; i++ {
		nrender[i] = make([]uint64, len(wd.render))
	}

	x0 := wd.Bounds().Min.X
	y0 := wd.Bounds().Min.Y
	Stride := wd.Bounds().Dx()
	ch := make(chan []uint64)

	renderf := func(r []uint64, c cell) {
		b := c.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r[x-x0+(y-y0)*Stride] += c.ValueAt(x, y)
			}
		}
		ch <- r
	}

	// Do separate renderings of dipoles
	for i := 0; i < n; i++ {
		go renderf(nrender[i], wd.dipoles[i].N)
	}

	// Whenever a goroutine finished rendering a dipole,
	// start again with a new dipole, until all dipoles
	// have been rendered.
	for i := n; i < len(wd.dipoles); i++ {
		go renderf(<-ch, wd.dipoles[i].N)
	}

	for i := 0; i < n; i++ {
		_ = <-ch
	}

	// Sum all of the renders
	r0 := nrender[0]
	for i := 1; i < n; i++ {
		ri := nrender[i]
		for j := 0; j < len(r0); j++ {
			r0[j] += ri[j]
		}
	}

	for i := 0; i < len(wd.render); i++ {
		wd.render[i] = uint16(r0[i])
	}
}

func (wd *WDMap) ValueAt(x, y int) (v uint16) {
	if r := wd.Bounds(); !(image.Point{x, y}.In(r)) {
		return
	} else {
		v = wd.render[x-r.Min.X+(y-r.Min.Y)*r.Dx()]
	}
	return
}

func (wd *WDMap) At(x, y int) color.Color {
	return color.Gray16{wd.ValueAt(x, y)}
}

// Splits the dipoles in WDMap in two, using n goroutines
func (wd *WDMap) SplitCells(n int) {
	if n <= 0 {
		n = 1
	}

	l := len(wd.dipoles)
	if l < n {
		n = l
	}

	ch := make(chan *dipole)
	for i := 0; i < n; i++ {
		go wd.makeMasks(i, ch)
	}
	for i := n; i < l; i++ {
		if c := <-ch; c != nil {
			wd.dipoles = append(wd.dipoles, *c)
		}
		go wd.makeMasks(i, ch)
	}
	for i := 0; i < n; i++ {
		if c := <-ch; c != nil {
			wd.dipoles = append(wd.dipoles, *c)
		}
	}

	// Try to avoid the heap from growing too much, because of
	// GC issues on 32 bit systems (bit cargo-cultish, I know)
	//runtime.GC()
}

func (wd *WDMap) makeMasks(idx int, ch chan *dipole) {

	Result := wd.dipoles[idx].N.Result
	//Too small to divide further
	if Result == nil {
		ch <- nil
		return
	}

	if Result.Mass() <= 0xFFFF {
		ch <- nil
		return
	}

	r := Result.Bounds()
	x0, y0 := Result.CM()
	x1, y1 := wd.dipoles[idx].S.Result.CM()

	// Calculate the weighted centre. Yes, these four lines
	// are the only difference between WDMap and DMap
	w0 := float64(Result.Mass())
	w1 := float64(wd.dipoles[idx].S.Result.Mass())

	cx := (x0*w1 + x1*w0) / (w0 + w1)
	cy := (y0*w1 + y1*w0) / (w0 + w1)

	// Find the slopes along x and y for the line that is the set of points
	// at equal distance from (x0, y0) and (x1, y1), then split along the
	// one that is least steep (which is always < 1 ).
	dx := x1 - x0
	dy := y1 - y0
	var h bool
	if math.Abs(dx) < math.Abs(dy) {
		h = true
		dy = -dx / dy
	} else if math.Abs(dy) < math.Abs(dx) {
		dx = -dy / dx
	} else {
		// If both centres of mass are at the same spot, due to symmetry
		// or homogenous density, split along the shortest axis.
		h = r.Dx() < r.Dy()
		dx = 0
		dy = 0
	}

	ma := density.NewMap(r)
	mb := density.NewMap(r)

	col := func(d int, dd, t float64) (c uint16) {
		if d < int(t) {
			c = 0xFFFF
		} else if d == int(t) {
			t -= float64(d)
			c = uint16(t * 0xFFFF)
		}
		return
	}
	var c uint16
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if h {
				t := cy - dy*(cx-float64(x))
				c = col(y, dy, t)
			} else {
				t := cx - dx*(cy-float64(y))
				c = col(x, dx, t)
			}
			ma.InitSet(x, y, c)
			mb.InitSet(x, y, 0xFFFF-c)
		}
	}

	dch := make(chan *density.Map)
	intersectf := func(m1, m2 *density.Map) {
		dch <- m1.CompactIntersect(m2)
	}
	go intersectf(ma, wd.dipoles[idx].N.Mask)
	go intersectf(mb, wd.dipoles[idx].N.Mask)
	nm1 := <-dch
	nm2 := <-dch

	if nm1 == nil || nm2 == nil {
		ch <- nil
		return
	}
	if nm1.Mass() == 0 || nm2.Mass() == 0 {
		ch <- nil
		return
	}
	wd.dipoles[idx].setMask(nm1)
	ch <- newDipole(wd.dipoles[idx].N.Source, wd.dipoles[idx].S.Source, nm2)
}

func NewWD(i image.Image, nd, sd density.Model, c uint) (wd *WDMap) {
	if c == 0 {
		c = 1
	}
	r := i.Bounds()
	wd = &WDMap{
		dipoles: make([]dipole, 1, c),
		render:  make([]uint16, r.Dx()*r.Dy()),
		rect:    r,
	}

	n := density.MapFrom(i, nd)
	s := density.MapFrom(i, sd)
	m := density.NewMap(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			m.InitSet(x, y, 0xFFFF)
		}
	}

	wd.dipoles[0] = *newDipole(n, s, m)
	return
}

type ColWDMap struct {
	R, G, B *WDMap
}

func NewColWD(i image.Image, c uint) (cwd *ColWDMap) {
	if c == 0 {
		c = 1
	}
	cwd = &ColWDMap{
		R: NewWD(i, density.RedDensity, density.NegRedDensity, c),
		G: NewWD(i, density.GreenDensity, density.NegGreenDensity, c),
		B: NewWD(i, density.BlueDensity, density.NegBlueDensity, c),
	}
	return
}

func (c *ColWDMap) ColorModel() color.Model {
	return color.RGBAModel
}

func (c *ColWDMap) Bounds() image.Rectangle {
	return c.R.Bounds()
}

func (c *ColWDMap) At(x, y int) color.Color {
	r := uint8(c.R.ValueAt(x, y) >> 8)
	g := uint8(c.G.ValueAt(x, y) >> 8)
	b := uint8(c.B.ValueAt(x, y) >> 8)
	return color.RGBA{r, g, b, 0xFF}
}

func (c *ColWDMap) SplitCells(n int) {
	c.R.SplitCells(n)
	c.G.SplitCells(n)
	c.B.SplitCells(n)
}

func (c *ColWDMap) Render(n int) {
	c.R.Render(n)
	c.G.Render(n)
	c.B.Render(n)
}
