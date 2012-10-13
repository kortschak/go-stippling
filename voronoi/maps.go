package voronoi

import (
	"code.google.com/p/go-stippling/density"
)

type maps struct {
	dmap *density.Map
	sumx *density.SumX
	sumy *density.SumY
	dsum *density.DSum
}

// mass(p0, p1) gives the mass as over the area in p0 and p1,
// as an int64 with 10-bit FPM.
func (m *maps) subMass(p0, p1 Point) (mass uint64) {
	x0 := int(p0.X >> fpmbits)
	y0 := int(p0.Y >> fpmbits)
	x1 := int(p1.X >> fpmbits)
	y1 := int(p1.Y >> fpmbits)
	fxp0 := (0x400 - (p0.X & 0x3FF)) & 0x400
	fyp0 := (0x400 - (p0.Y & 0x3FF)) & 0x400
	fxp1 := p1.X & 0x3FF
	fyp1 := p1.Y & 0x3FF

	// top-left corner and bottem-left corner
	if fxp0 != 0 {
		if fyp0 != 0 {
			mass += (fxp0 * fyp0 * m.dmap.ValueAt(x0, y0)) >> fpmbits
		} else {
			mass += fxp0 * m.dmap.ValueAt(x0, y0)
		}

		if fyp1 != 0 {
			mass += (fxp0 * fyp1 * m.dmap.ValueAt(x0, y1)) >> fpmbits
		}
	} else {
		if fyp0 != 0 {
			mass += fyp0 * m.dmap.ValueAt(x0, y0)
		} else {
			mass += m.dmap.ValueAt(x0, y0) << fpmbits
		}
		if fyp1 != 0 {
			mass += fyp1 * m.dmap.ValueAt(x0, y1)
		}
	}

	// top right corner and bottom-right corner
	if fxp1 != 0 {
		if fyp0 != 0 {
			mass += (fxp1 * fyp0 * m.dmap.ValueAt(x1, y0)) >> fpmbits
		}
		if fyp1 != 0 {
			mass += (fxp1 * fyp1 * m.dmap.ValueAt(x1, y1)) >> fpmbits
		}
	}

	// top factional row, without fractional corners
	if fyp0 != 0 {
		mass += fyp0 * (m.sumx.ValueAt(x1, y0) - m.sumx.ValueAt(x0, y0))
	}

	// bottom fractional row without fractional corners
	if fyp1 != 0 {
		mass += fyp1 * (m.sumx.ValueAt(x1, y1) - m.sumx.ValueAt(x0, y1))
	}

	// left-most fractional column, without fractional corners
	if fxp0 != 0 {
		mass += fxp0 * (m.sumy.ValueAt(x0, y1) - m.sumy.ValueAt(x0, y0))
	}

	// right-most fractional column, without fractional corners
	if fxp1 != 0 {
		mass += fxp1 * (m.sumy.ValueAt(x1, y1) - m.sumy.ValueAt(x1, y0))
	}

	// center area, without fractional borders
	mass += m.dsum.AreaSum(x0, y0, x1, y1) << fpmbits
}

// Gives the centre of mass of the area enclosed by p0 and p1
func (m *maps) cm(p0, p1 Point) (c Point) {
	var mass, dm uint64

	x0 := int(p0.X >> fpmbits)
	y0 := int(p0.Y >> fpmbits)
	x1 := int(p1.X >> fpmbits)
	y1 := int(p1.Y >> fpmbits)
	fxp0 := (0x400 - (p0.X & 0x3FF)) & 0x400
	fyp0 := (0x400 - (p0.Y & 0x3FF)) & 0x400
	fxp1 := p1.X & 0x3FF
	fyp1 := p1.Y & 0x3FF

	// Finding WX. I hate FPM... Self-documenting in
	// an attempt keep overview of structure.

	// Leftmost column first.

	// Check if top-left corner is a fraction in y-axis
	if fyp0 != 0 {
		dm = fyp0 * m.dmap.ValueAt(x0, y0)
	} else {
		dm = m.dmap.ValueAt(x0, y0) << fpmbits
	}

	// Leftmost column without corners
	dm += (m.sumy.ValueAt(x0, y1) - m.sumy.ValueAt(x0, y0)) << fpmbits

	// check if bottom-left corner is a fraction in y-axis
	if fyp1 != 0 {
		dm += fyp1 * m.dmap.ValueAt(x0, y1)
	}

	// check is leftmost column + corners are a fraction
	// in the x axis, last correction, add everything to
	// c.X and mass
	if fxp0 != 0 {
		c.X = (uint64(x0) * fxp0 * dm) >> fpmbits
		mass = (fxp0 * dm) >> fpmbits
	} else {
		c.X = uint64(x0) * dm
		mass = dm
	}

	//Middle columns, without left and right columns or bottom row

	// Correct top-most row without corners, if a fraction
	if fyp0 != 0 {
		for x := x0 + 1; x < x1; x++ {
			dm = m.dmap.ValueAt(x, y0) * fyp0
			c.X += uint64(x) * dm
			mass += dm
		}
	} else {
		y0--
	}

	for x := x0 + 1; x < x1; x++ {
		dm = (m.sumy.ValueAt(x, y1) - m.sumy.ValueAt(x, y0))
		c.X += uint64(x) * dm
		mass += dm
	}

	if fyp0 == 0 {
		y0++
	}

	// Bottom row, excluding corners
	if fyp1 != 0 {
		for x := x0 + 1; x < x1; x++ {
			dm = m.dmap.ValueAt(x, y1) * fyp1
			c.X += uint64(x) * dm
			mass += dm
		}
	}

	// Check if rightmost column is a fraction in the x-axis
	if fxp1 != 0 {
		if fyp0 != 0 {
			dm = fyp0 * m.dmap.ValueAt(x1, y0)
		}

		// Rightmost column without corners
		dm += (m.sumy.ValueAt(x1, y1) - m.sumy.ValueAt(x1, y0)) << fpmbits

		// check if bottom-right corner is a fraction in y-axis
		if fyp1 != 0 {
			dm += fyp1 * m.dmap.ValueAt(x1, y1)
		}

		// Correct for fraction, add to c.X and mass
		c.X = (uint64(x1) * fxp1 * dm) >> fpmbits
		mass = (fxp1 * dm) >> fpmbits
	}

	// Find WY. Similar procedure to WX, without the mass part.

	// Topmost row first.

	// Check if top-left corner is a fraction in x-axis
	if fxp0 != 0 {
		dm = fxp0 * m.dmap.ValueAt(x0, y0)
	} else {
		dm = m.dmap.ValueAt(x0, y0) << fpmbits
	}

	// Topmost row without corners
	dm += (m.sumx.ValueAt(x1, y0) - m.sumx.ValueAt(x0, y0)) << fpmbits

	// check if top-right corner is a fraction in x-axis
	if fxp1 != 0 {
		dm += fyp1 * m.dmap.ValueAt(x1, y0)
	}

	// check if topmost row + corners are a fraction
	// in the y axis, last correction, add everything to
	// c.Y and mass
	if fyp0 != 0 {
		c.Y = (uint64(y0) * fyp0 * dm) >> fpmbits
	} else {
		c.Y = uint64(y0) * dm
	}

	// Middle rows, without top and bottom rows and rightmost column

	// Correct left-most column without corners, if a fraction
	if fxp0 != 0 {
		for y := y0 + 1; y < y1; y++ {
			c.Y += m.dmap.ValueAt(x0, y) * fxp0
		}
	} else {
		x0--
	}

	for y := y0 + 1; y < y1; y++ {
		c.Y += uint64(y) * (m.sumx.ValueAt(x0, y) - m.sumx.ValueAt(x0, y))
	}

	if fxp0 == 0 {
		x0++
	}

	// Rightmost column, excluding corners
	if fxp1 != 0 {
		for y := y0 + 1; y < y1; y++ {
			c.Y += m.dmap.ValueAt(x0, y) * fxp1
		}
	}

	// Check if bottom row is a fraction in the y-axis
	if fyp1 != 0 {
		if fxp0 != 0 {
			dm = fxp0 * m.dmap.ValueAt(x0, y1)
		}

		// Rightmost column without corners
		dm += (m.sumx.ValueAt(x1, y1) - m.sumx.ValueAt(x0, y1)) << fpmbits

		// check if bottom-right corner is a fraction in y-axis
		if fxp1 != 0 {
			dm += fxp1 * m.dmap.ValueAt(x1, y1)
		}

		// Correct for fraction, add to c.Y
		c.Y = (uint64(x1) * fyp1 * dm) >> fpmbits
	}

	// correct X and Y for mass
	c.X /= mass
	c.Y /= mass
	return
}
