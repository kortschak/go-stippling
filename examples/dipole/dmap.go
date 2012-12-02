package main

// dipmap takes an image and converts it into two density maps: 
// a positive and negative version. The centres of mass of 
// these are then used as two poles. Using these as generators,
// two voronoi cells are generated within the boundaries. For
// each cell, this process can then be repeated: determine the
// positive/negative centres of mass of the cell, use these as
// generators for two new voronoi cells, bound within the
// parent voronoi cell. Then repeat for the new voronoi cells
//
// dipmap implements the Gray16 image.Image interface, with each
// cell the color of the average density of the area within the
// positive map covered by the cell.

import (
	"code.google.com/p/go-stippling/density"
	//"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
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

type DMap struct {
	dipoles []dipole
	render  []uint16
	rect    image.Rectangle
}

func (dm *DMap) Copy(a *DMap) {
	copy(dm.dipoles, a.dipoles)
	copy(dm.render, a.render)
	dm.rect = a.rect
}

func (dm *DMap) ColorModel() color.Model {
	return color.Gray16Model
}

func (dm *DMap) Bounds() image.Rectangle {
	return dm.rect
}

//Render dipoles using n goroutines
func (dm *DMap) Render(n int) {
	if n <= 0 {
		n = 1
	}
	if len(dm.dipoles) < n {
		n = len(dm.dipoles)
	}

	nrender := make([][]uint64, n)

	for i := 0; i < n; i++ {
		nrender[i] = make([]uint64, len(dm.render))
	}

	x0 := dm.Bounds().Min.X
	y0 := dm.Bounds().Min.Y
	Stride := dm.Bounds().Dx()
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
		go renderf(nrender[i], dm.dipoles[i].N)
	}

	// Whenever a goroutine finished rendering a dipole,
	// start again with a new dipole, until all dipoles
	// have been rendered.
	for i := n; i < len(dm.dipoles); i++ {
		go renderf(<-ch, dm.dipoles[i].N)
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

	for i := 0; i < len(dm.render); i++ {
		dm.render[i] = uint16(r0[i])
	}
}

func (dm *DMap) ValueAt(x, y int) (v uint16) {
	if r := dm.Bounds(); !(image.Point{x, y}.In(r)) {
		return
	} else {
		v = dm.render[x-r.Min.X+(y-r.Min.Y)*r.Dx()]
	}
	return
}

func (dm *DMap) At(x, y int) color.Color {
	return color.Gray16{dm.ValueAt(x, y)}
}

// Splits the dipoles in DMap in two, using n goroutines
func (dm *DMap) SplitCells(n int) {
	if n <= 0 {
		n = 1
	}

	l := len(dm.dipoles)
	if l < n {
		n = l
	}

	ch := make(chan *dipole)
	for i := 0; i < n; i++ {
		go dm.makeMasks(i, ch)
	}
	for i := n; i < l; i++ {
		if c := <-ch; c != nil {
			dm.dipoles = append(dm.dipoles, *c)
		}
		go dm.makeMasks(i, ch)
	}
	for i := 0; i < n; i++ {
		if c := <-ch; c != nil {
			dm.dipoles = append(dm.dipoles, *c)
		}
	}
	// Try to avoid the heap from growing too much, because of
	// GC issues on 32 bit systems (bit cargo-cultish, I know)
	runtime.GC()
}

func (dm *DMap) makeMasks(idx int, ch chan *dipole) {

	Result := dm.dipoles[idx].N.Result
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
	x1, y1 := dm.dipoles[idx].S.Result.CM()

	cx := (x0 + x1) / 2
	cy := (y0 + y1) / 2

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
	go intersectf(ma, dm.dipoles[idx].N.Mask)
	go intersectf(mb, dm.dipoles[idx].N.Mask)
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
	dm.dipoles[idx].setMask(nm1)
	ch <- newDipole(dm.dipoles[idx].N.Source, dm.dipoles[idx].S.Source, nm2)
}

func NewDMap(i image.Image, nd, sd density.Model, c uint) (dm *DMap) {
	if c == 0 {
		c = 1
	}
	r := i.Bounds()
	dm = &DMap{
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

	dm.dipoles[0] = *newDipole(n, s, m)
	return
}

type ColDMap struct {
	R, G, B *DMap
}

func NewColDMap(i image.Image, c uint) (cdm *ColDMap) {
	if c == 0 {
		c = 1
	}
	cdm = &ColDMap{
		R: NewDMap(i, density.RedDensity, density.NegRedDensity, c),
		G: NewDMap(i, density.GreenDensity, density.NegGreenDensity, c),
		B: NewDMap(i, density.BlueDensity, density.NegBlueDensity, c),
	}
	return
}

func (c *ColDMap) ColorModel() color.Model {
	return color.RGBAModel
}

func (c *ColDMap) Bounds() image.Rectangle {
	return c.R.Bounds()
}

func (c *ColDMap) At(x, y int) color.Color {
	r := uint8(c.R.ValueAt(x, y) >> 8)
	g := uint8(c.G.ValueAt(x, y) >> 8)
	b := uint8(c.B.ValueAt(x, y) >> 8)
	return color.RGBA{r, g, b, 0xFF}
}

func (c *ColDMap) SplitCells(n int) {
	c.R.SplitCells(n)
	c.G.SplitCells(n)
	c.B.SplitCells(n)
}

func (c *ColDMap) Render(n int) {
	c.R.Render(n)
	c.G.Render(n)
	c.B.Render(n)
}
