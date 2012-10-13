package voronoi

// lineEq represents a line in the form the equation:
//
//	 y(x) = (a0/a1)*x + b
//
// or, depending on the context:
//
//   x(y) = (a0/a1)*y + b
//
// Used to represent growing boundaries of cells and
// predict their intersections, before converting them
// to their "final" form with a start and end point.
type lineEq struct {
	// if the line is orthogonal to the axis, a1 = 0
	a0, a1, b int64
}

// Boundary line
type bline struct {
	l0, l1 lineEq
	c0, c1 *cell
}

// Update boundaries of weighed cells based on current
// weight of generators and their position.
func (d *Diagram) sweep() {
	activeLines := make([]bline, 0, len(d.xsorted))

	// General procedure: look at the intersections
	// of the active lines, and the next generator
	// Jump to the nearest of these. If intersection,
	// update active lines and relevant cells.
	// If new generator, insert generator, then try
	// to backtrack through the already existing
	// active and "final" lines, updating them as
	// necessary. It is quite similar to casting a
	// ray, actually.

	// Sweep in X direction first

}

// calculate point of intersection between two lines, using
// the following equations: 
// 
//  y  = x*a0/a1  + b = x*c0/c1  + d
//
// This obviously only makes sense if:
//
//	 a0*c1 - a1*c0 	!=	0
//   a1*c1 			!=	0 
//
// Which is a condition implying parallel lines
//
// Note that (ignoring y for a moment) this equation can
// be rewritten as:
//
//   x*((a0/a1) - (c0/c1)) = d - b
//
//   x  = (d - b) / ((a0/a1) - (c0/c1))
//	    = (d - b) * a1*c1 / (a0*c1 - a1*c0)
//
// Now to prevent overflow: 10 bits FPM on signed 64 bits integers, 
// assume heigth/width is as most 32K pixels, or 15+10=25 bits set
// at most, which gives 13 spare bits of headroom for minimising
// rounding errors, or:
//
//   x  = ((d-b) * ((a1*c1) >> 12)) / ((a0*c1-a1*c0) >> 12 )
//
// in bits used:
//
//   25 = (  25  + ((25+25)  - 12)) - ((  25 + 25  )  - 12 ) 
//      = (  25  +         38     ) - (         38         )
//
// Unless I'm gravely mistaken, this should minimise rounding errors.
//
// As for y:
//
//   y  = x*a0/a1 + b    = x*c0/c1 + d
//   x  = y*a1/a0 - b/a0 = y*c1/c0 - d/c0
//
// Which again can be rewritten as:
//
//   y*(a1/a0 - c1/c0) = b/a0 - d/c0
//   y = (b/a0 - d/c0) / (a1/a0 - c1/c0)
//   y = ((b*c0 - d*a0)/(a0*c0)) / ((a1*c0 - c1*a0)/(a0*c0))
//   y = (b*c0 - d*a0) / (a1*c0 - c1*a0)
//
// That is, provided c0 or a0 isn't zero. If a0 is zero, y = b,
// if c0 is zero, y = d. We don't have to worry about overflow,
// since the x-equation will overflow before the y-equation does.
func intersectX(l0, l1 lineEq) (x, y int64, theyintersect bool) {
	if l0.a0*l1.a1-l0.a1*l1.a0 == 0 || l0.a1 == 0 || l1.a1 == 0 {
		return
	}

	theyintersect = true

	// See comment above for explanation of this monstrosity
	//   x  = ((d-b) * ((a1*c1) >> 12)) / ((a0*c1-a1*c0) >> 12 )
	x = ((l1.b - l0.b) * ((l0.a1 * l1.a1) >> 12)) / ((l0.a0*l1.a1 - l0.a1*l1.a0) >> 12)

	if l0.a0 == 0 {
		y = l0.b
	} else if l1.a0 == 0 {
		y = l1.b
	} else {
		y = (l0.b*l1.a0 - l1.b*l0.a0) / (l0.a1*l1.a0 - l1.a1*l0.a0)
	}
}
