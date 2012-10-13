package voronoi

import (
	"code.google.com/p/go-stippling/density"
	"image"
)

// Almost identical to image.Point, but using 64 bit integer FPM,
// with the fractional part being 10 bits - 1024 by 1024 subpixel
// precision should be enough for most intents and purposes.
//
// Note that summing over an area gives a total of (2^10)*(2^10) 
// subpixels, in other words: the mass' fraction can have up to
// 20 bit "precision" if necessary.
type Point struct {
	X, Y uint64
}

// A Boundary is saved as a starting and ending point,
// and a pointer to the neighbouring Voronoi cell.
type Boundary struct {
	p0, p1    Point
	neighbour *cell
}

func (b Boundary) toXLine(l lineEq) {
	if b.p0.X > b.p1.X {
		b.p0, b.p1 = b.p1, b.p0
	}

	l.a0 = b.p1.Y - b.p0.Y
	l.a1 = b.p1.X - b.p0.X
	if l.a1 != 0 {
		l.b = b.p0.Y - b.p0.X*l.a0/l.a1
	}
}

func (b Boundary) toYLine(l lineEq) {
	if b.p0.Y > b.p1.Y {
		b.p0, b.p1 = b.p1, b.p0
	}

	l.a0 = b.p1.X - b.p0.X
	l.a1 = b.p1.Y - b.p0.Y
	if l.a1 != 0 {
		l.b = b.p0.X - b.p0.Y*l.a0/l.a1
	}
}

type cell struct {
	Point
	up, down, left, right []Boundary
	mass, nmass           uint64
}

type Diagram struct {
	xsorted, ysorted []*cell
	maps
}

func NewDiagram(i image.Image, m density.Model, ncells uint64) (d *Diagram) {
	d.maps.dmap = density.MapFrom(i, m)
	d.maps.sumx = density.SumXFrom(i, m)
	d.maps.sumy = density.SumYFrom(i, m)
	d.maps.dsum = density.DSumFrom(i, m)

	d.xsorted = make([]*cell, ncells)
	d.ysorted = make([]*cell, ncells)

	// Initial guess as to where the generators should be put.
	cellchan := make(chan Point)
	rect := d.maps.dmap.Bounds()
	// Theoretically, rect.Min can be negative, but since we open
	// the images ourselves we know that will never happen, and
	// in fact it is guaranteed that the values will be zero.
	p0 := Point{uint64(rect.Min.X << 12), uint64(rect.Min.Y << 12)}
	p1 := Point{uint64(rect.Max.X << 12), uint64(rect.Max.Y << 12)}
	go guess(d.maps, p0, p1, ncells, cellchan)
	nillbound := make([]Boundary, 0, 0)
	for i := 0; uint64(i) < ncells; i++ {
		p := &cell{
			Point: <-cellchan,
			up:    nillbound,
			down:  nillbound,
			left:  nillbound,
			right: nillbound,
			mass:  0,
			nmass: 0,
		}

		// Not the most efficient way to sort, but it works
		for j := 0; j < i; j++ {
			if p.X > d.xsorted[j].X {
				for k := i; k > j; k-- {
					d.xsorted[k] = d.xsorted[k-1]
				}
				d.xsorted[j] = p
				j = i
			}
		}
		for j := 0; j < i; j++ {
			if p.Y > d.ysorted[j].Y {
				for k := i; k > j; k-- {
					d.ysorted[k] = d.ysorted[k-1]
				}
				d.ysorted[j] = p
				j = i
			}
		}
	}

	//TODO: implement sweepline algorithm that gives cells their
	//		initial boundaries and mass values.
	return
}

// The guessing algorithm is based on the simple observation that once
// in equilibrium, all generators have mass equal to:
//
//   total mass / total generators
//
// Hence, the following should result in a decent initial guess: Take
// the density map, split it in two along the longest axis such that
// the cells can be divided as evenly as possible among the submaps,
// and the mass is divided proportionaly to the number of cells being
// divided. Repeat this process with the submaps, until there is only
// one cell left in a submap, which will have average mass. To get an
// early start on Floyd relaxation, the center of the generator is
// then put on the centre of mass of this submap.
//
// Note that we can trivially make this algorithm concurrent.
func guess(m maps, p0, p1 Point, ncells uint64, c chan Point) {
	if ncells == 1 {
		c <- m.cm(p0, p1)
	} else {
		n0 := ncells >> 1
		n1 := ncells - n0
		mass := m.subMass(p0, p1)
		targetmass := n0 * mass / ncells

		dx := p1.X - p0.X
		dy := p1.Y - p0.Y
		dp := p1

		// divide along longest axis
		if dx < dy {
			y := dy >> 1
			dp.Y = p0.Y + y
			dmass := m.subMass(p0, dp)
			for dmass != targetmass {
				dy >>= 1
				if dmass < targetmass {
					y += dy
				} else {
					y -= dy
				}
				dp.Y = p0.Y + y
				dmass := m.subMass(p0, dp)
			}
			go guess(m, p0, dp, n0, c)
			dp.X = p0.X
			go guess(m, dp, p1, n1, c)
		} else {
			x := dx >> 1
			dp.X = p0.X + x
			dmass := m.subMass(p0, dp)
			for dmass != targetmass {
				dx >>= 1
				if dmass < targetmass {
					x += dx
				} else {
					x -= dx
				}
				dp.X = p1.X - x
				dmass := m.subMass(p0, dp)
			}
			go guess(m, p0, dp, n0, c)
			dp.Y = p0.Y
			go guess(m, dp, p1, n1, c)
		}
	}
	return
}
