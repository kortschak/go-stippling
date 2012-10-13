package main

// WDMap takes an image and converts it into two density maps (usually a positive and 
// negative version). The centres of mass of these are then used as the base for a  
// dipole. Using the poles of the dipole, two voronoi dipoles are generated. For each
// voronoi cell, this process can then be repeated, each time resulting in two new
// voronoi dipoles within the parent voronoi cell, with their generators on the centres 
// of mass of the parent cell.
//
// WDMap implements the Gray16 image.Image interface, with each cell the color of
// the average density of the area within the first map covered by the cell

import (
	"code.google.com/p/go-stippling/density"
	"image"
	"image/color"
	"math"
	"runtime"
)

type dipole struct {
	//North and South
	N, S voronoi.Cell
}

func newDipole(n, s, m *density.Map) (d *dipole) {
	d = &dipole{
		N: voronoi.Cell{S: n, M: m},
		S: voronoi.Cell{S: s, M: m},
	}
	d.N.ChangedMaps()
	d.S.ChangedMaps()
	return
}

func (d *dipole) setMask(m *density.Map) {
	d.N.M = m
	d.S.M = m
	d.N.ChangedMaps()
	d.S.ChangedMaps()
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
	renderf := func(r []uint64, c voronoi.Cell) {
		b := c.Bounds()
		mass := c.Mass()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if mv := uint64(c.M.ValueAt(x, y)); mv > 0 {
					r[x-x0+(y-y0)*Stride] += mv * mass
				}
			}
		}
		ch <- r
	}

	for i := 0; i < n; i++ {
		go renderf(nrender[i], wd.dipoles[i].N)
	}
	for i := n; i < len(wd.dipoles); i++ {
		go renderf(<-ch, wd.dipoles[i].N)
	}

	for i := 0; i < n; i++ {
		_ = <-ch
	}

	close(ch)
	r0 := nrender[0]
	for i := 1; i < n; i++ {
		ri := nrender[i]
		for j := 0; j < len(r0); j++ {
			r0[j] += ri[j]
		}
	}
	for i := 0; i < len(wd.render); i++ {
		wd.render[i] = uint16(r0[i] / 0xFFFF)
	}

	r0 = nil
	nrender = nil
	ch = nil
	runtime.GC()
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
	close(ch)
	ch = nil
	// Try to avoid the heap from growing too much, because of
	// GC issues on 32 bit systems (bit cargo-cultish, I know)
	runtime.GC()
}

func (wd *WDMap) makeMasks(idx int, ch chan *dipole) {

	M := wd.dipoles[idx].N.M
	if M.Mass() <= 0xFFFF {
		ch <- nil
		return
	}

	r := M.Bounds()
	x0, y0 := wd.dipoles[idx].N.CM()
	x1, y1 := wd.dipoles[idx].S.CM()

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
	cif := func(m1, m2 *density.Map) {
		dch <- density.CompactIntersect(m1, m2)
	}
	go cif(ma, M)
	go cif(mb, M)
	ma = <-dch
	mb = <-dch
	close(dch)

	if ma.Mass() == 0 || mb.Mass() == 0 {
		ch <- nil
		return
	}
	wd.dipoles[idx].setMask(ma)
	ch <- newDipole(wd.dipoles[idx].N.S, wd.dipoles[idx].S.S, mb)
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

/*
// cell - has pointer to positive density map, 
// negative density map and mask. 
type cell struct {
	p, n, m *density.Map
}

type WDMap struct {
	p, n   density.Map
	dipoles  []cell
	render []uint16
}

func (wd *WDMap) Copy(a *WDMap) {
	wd.src = a.src
	wd.dipoles = make([]cell, len(a.dipoles), cap(a.dipoles))
	copy(wd.dipoles, a.dipoles)
}

func (wd *WDMap) ColorModel() color.Model {
	return color.Gray16Model
}

func (wd *WDMap) Bounds() image.Rectangle {
	return wd.src.p.Bounds()
}

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
	f := func(r []uint64, c cell) {
		b := c.m.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				mv := uint64(c.m.ValueAt(x, y))
				if mv > 0 {
					r[x-x0+(y-y0)*Stride] += mv * ((c.p.Mass() << 16) / c.m.Mass())
				}
			}
		}
		ch <- r
	}

	for i := 0; i < n; i++ {
		go f(nrender[i], wd.dipoles[i])
	}
	for i := n; i < len(wd.dipoles); i++ {
		go f(<-ch, wd.dipoles[i])
	}

	for i := 0; i < n; i++ {
		_ = <-ch
	}
	close(ch)
	r0 := nrender[0]
	for i := 1; i < n; i++ {
		ri := nrender[i]
		for j := 0; j < len(r0); j++ {
			r0[j] += ri[j]
		}
	}
	for i := 0; i < len(wd.render); i++ {
		wd.render[i] = uint16(r0[i] >> 16)
	}
	r0 = nil
	nrender = nil
	ch = nil
	runtime.GC()
}

func (wd *WDMap) ValueAt(x, y int) uint16 {
	r := wd.Bounds()
	return wd.render[x-r.Min.X+(y-r.Min.Y)*r.Dx()]
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
	ch := make(chan *cell)

	for i := 0; i < n; i++ {
		go wd.makeMasks(wd.dipoles[i].p, wd.dipoles[i].n, wd.dipoles[i].m, ch)
	}
	for i := n; i < l; i++ {
		nc := <-ch
		if nc != nil {
			wd.dipoles = append(wd.dipoles, *nc)
		}
		go wd.makeMasks(wd.dipoles[i].p, wd.dipoles[i].n, wd.dipoles[i].m, ch)
	}
	for i := 0; i < n; i++ {
		nc := <-ch
		if nc != nil {
			wd.dipoles = append(wd.dipoles, *nc)
		}
	}
	close(ch)
	ch = nil
	// Try to avoid the heap from growing too much, because of
	// GC issues on 32 bit systems (bit cargo-cultish, I know)
	runtime.GC()
}

func (wd *WDMap) makeMasks(p, n, m *density.Map, ch chan *cell) {

	if m.Mass() <= 0xFFFF {
		ch <- nil
		return
	}

	r := m.Bounds()
	x0, y0 := p.CM()
	x1, y1 := n.CM()

	cx := (x0 + x1) / 2
	cy := (y0 + y1) / 2

	// Find the slopes along x and y for the line that is the set of 
	// points at equal distance from (x0, y0) and (x1, y1) and split
	// along the gentlest one.
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
	cif := func(m1, m2 *density.Map) {
		dch <- density.CompactIntersect(m1, m2)
	}
	go cif(ma, m)
	go cif(mb, m)
	ma = <-dch
	mb = <-dch
	close(dch)
	dch = nil

	if ma.Mass() == 0 || mb.Mass() == 0 {
		ch <- nil
		return
	}

	*m = *ma
	p.Copy(density.CompactIntersect(ma, wd.src.p))
	n.Copy(density.CompactIntersect(ma, wd.src.n))
	ch <- &cell{
		density.CompactIntersect(mb, wd.src.p),
		density.CompactIntersect(mb, wd.src.n),
		mb,
	}
	return
}

func NewWD(i image.Image, d density.Model, c uint) (wd *WDMap) {
	if c == 0 {
		c = 1
	}
	r := i.Bounds()
	wd = &WDMap{
		src:    cell{},
		dipoles:  make([]cell, 1, c),
		render: make([]uint16, r.Dx()*r.Dy()),
	}

	ps := density.MapFrom(i, d)
	ns := density.NewMap(r)
	wd.dipoles[0].m = density.NewMap(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ns.InitSet(x, y, 0xFFFF-ps.ValueAt(x, y))
			wd.dipoles[0].m.InitSet(x, y, 0xFFFF)
		}
	}
	wd.src.p = density.CompactIntersect(ps, wd.dipoles[0].m)
	wd.src.n = density.CompactIntersect(ns, wd.dipoles[0].m)
	wd.dipoles[0].p = density.CompactIntersect(ps, wd.dipoles[0].m)
	wd.dipoles[0].n = density.CompactIntersect(ns, wd.dipoles[0].m)
	copy(wd.render, wd.src.p.Values)
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
		R: NewWD(i, density.RedDensity, c),
		G: NewWD(i, density.GreenDensity, c),
		B: NewWD(i, density.BlueDensity, c),
	}
	return
}

func (c *ColWDMap) ColorModel() color.Model {
	return color.RGBAModel
}

func (c *ColWDMap) Bounds() image.Rectangle {
	return c.R.src.p.Bounds()
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
*/
