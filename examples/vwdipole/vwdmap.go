// vwdipmap is almost equivalent to wdipmap, except for one
// crucial thing: it uses SumMaps and SumMasks, greatly
// reducing memory footprint.
package main

import (
	"code.google.com/p/go-stippling/density"
	"code.google.com/p/intmath/i64"
	//"fmt"
	"image"
	//"image/color"
	//"math"
	//"runtime"
)

const (
	FPM = 8
)

type dipoleMask struct {
	//North and South
	N, S *density.Sum
	Mask *density.SumMask
}

func (d *dipoleMask) Bounds() image.Rectangle {
	return d.Mask.X.Bounds()
}

// This function returns two dipoleMasks that divide the orignal dipole
// along the weighted dividing line between the north and south poles.
func (dp *dipoleMask) Split() (dn, ds *dipoleMask) {

	r := dp.Bounds()

	// Use 8 bit fpm to represent the weight and weighted coordinates
	// of the poles. Note that the latter don't necessarily map to r,
	// that is: the top left is (0,0), not (r.Min.X, r.Min.Y).
	// This simplifies a number of calculations, and helps avoid
	// overflow, but has to be corrected for at the end.
	wn := int64(dp.Mask.X.MaskedMass(&dp.N.X))
	xn := int64(dp.Mask.Y.Wx(&dp.N.Y)<<FPM) / wn
	yn := int64(dp.Mask.X.Wy(&dp.N.X)<<FPM) / wn
	ws := int64(dp.Mask.X.MaskedMass(&dp.S.X))
	xs := int64(dp.Mask.Y.Wx(&dp.S.Y)<<FPM) / ws
	ys := int64(dp.Mask.X.Wy(&dp.S.X)<<FPM) / ws

	// calculate the weighted centrepoint.
	var cx, cy int64
	if wn+ws > 0 {
		cx = (xn*ws + xs*wn) / (wn + ws)
		cy = (yn*ws + ys*wn) / (wn + ws)
	} else {
		// No mass = centre of mass is in the middle. Actually should
		// be in the middle of the mask instead, but I'm lazy.
		cx = int64(r.Dx()) << 7
		cy = int64(r.Dy()) << 7
	}

	// Here (dx,dy) represent the vector from (xn,yn) to (xs,ys), which
	// is the vector from N to S. What we actually need is the vector
	// of the dividing line. We will correct for this later on.
	dx := xs - xn
	dy := ys - yn

	var steep bool
	if dx == 0 && dy == 0 {
		// If both centres of mass are at the same spot, due to symmetry
		// or homogenous density, split along the shortest axis.
		steep = r.Dx() > r.Dy()
	} else {
		// This might seem counterintuitive, but remember that
		// the dividing line is at a right angle with (dx,dy)
		steep = i64.Abs(dx) > i64.Abs(dy)
	}

	// Rotate dx and dy to represent the vector of the dividing line.
	// We define that if steep, this vector must be from top to
	// bottom, and if not from left to right.
	// Remember: (0,0) is top-left, positive x goes to the right,
	// positive y downwards.
	if dx*dy > 0 {
		dx, dy = i64.Abs(dy), i64.Abs(dx)
	} else if steep {
		dx, dy = -i64.Abs(dy), i64.Abs(dx)
	} else {
		dx, dy = i64.Abs(dy), -i64.Abs(dx)
	}

	// General idea:
	// Using the dividing line, we're going to create two new masks
	// for SumX and SumY. After this we intersect these with the
	// existing masks to create the final masks for the split cells.

	// First, the dividing line.
	// Euclidean geometry to the rescue! Use congruence to test where
	// the dividing line crosses the bounding box, and turn that into
	// a vector.  We recycle (xn,yn) as the starting point, and
	// (xs, ys) as the ending point describing the dividing vector.
	{
		maxX := int64(r.Dx()) << FPM
		maxY := int64(r.Dy()) << FPM

		// Damn, this requires some mental juggling
		// Better do thorough tests on it later...
		if steep { // by definition dy >= 0
			if dy != 0 && dx != 0 {
				xn = (cx*dy - cy*dx) / dy
				if xn < 0 { // dx > 0
					xn = 0
					yn = (cy*dx - cx*dy) / dx
				} else if xn > maxX { // dx < 0
					xn = maxX
					yn = (cy*dx - (cx-maxX)*dy) / dx
				} else {
					yn = 0
				}

				xs = (cx*dy + (maxY-cy)*dx) / dy
				if xs < 0 { // dx < 0
					xs = 0
					ys = (cy*dx - cx*dy) / dx
				} else if xs > maxX { // dx > 0
					xs = maxX
					ys = (cy*dx - (cx-maxX)*dy) / dx
				} else {
					ys = maxY
				}
			} else {
				xn, yn = cx, 0
				xs, ys = cx, maxY
			}
		} else { // by definition dx >= 0
			if dy != 0 && dx != 0 {
				yn = (cy*dx - cx*dy) / dx
				if yn < 0 { // dy > 0
					xn = (cx*dy - cy*dx) / dy
					yn = 0
				} else if yn > maxY { // dy < 0
					xn = (cx*dy - (cy-maxY)*dx) / dy
					yn = maxY
				}

				ys = (cy*dx + (maxX-cx)*dy) / dx
				if ys < 0 { // dy < 0
					xs = (cx*dy - cy*dx) / dy
					ys = 0
				} else if ys > maxY { // dy > 0
					xs = (cx*dy - (cy-maxY)*dx) / dy
					ys = maxY
				}
			} else {
				xn, yn = 0, cy
				xs, ys = maxX, cy
			}
		}

		// Now we create two new bounding boxes based on this dividing vector.
		var rn, rs image.Rectangle

		// Similar to the dividing vector:
		// If steep, rn = left, rs = right. If not steep, rn = top, rs = bottom.
		// Again, I would be surprised if I didn't make
		// a mistake somewhere here... test later

		// Also, this assumes the coordinates fit into a 56 bit signed number.
		// That's probably a safe assumption to make though.
		if steep {
			if dx < 0 {
				//rn.Min.X = 0
				rn.Max.X = int(xn)
				//rn.Min.Y = 0
				rn.Max.Y = int(ys)

				rs.Min.X = int(xs)
				rs.Max.X = maxX
				rs.Min.Y = int(yn)
				rs.Max.Y = maxY
			} else {
				//rn.Min.X = 0
				rn.Max.X = int(xs)
				rn.Min.Y = int(yn)
				rn.Max.Y = maxY

				rs.Min.X = int(xn)
				rs.Max.X = maxX
				//rs.Min.Y = 0
				rs.Max.Y = int(ys)
			}
		} else {
			if dy < 0 {
				//rn.Min.X = 0
				rn.Max.X = int(xs)
				//rn.Min.Y = 0
				rn.Max.Y = int(yn)

				rs.Min.X = int(xn)
				rs.Max.X = maxX
				rs.Min.Y = int(ys)
				rs.Max.Y = maxY

			} else {
				rn.Min.X = int(xn)
				rn.Max.X = maxX
				//rn.Min.Y = 0
				rn.Max.Y = int(ys)

				//rs.Min.X = 0
				rs.Max.X = int(xs)
				rs.Min.Y = int(yn)
				rs.Max.Y = maxY
			}
		}
	}
	// Scale down to non-FPM boundaries
	rn.Min.X = (rn.Min.X >> FPM) + r.Min.X
	rn.Min.Y = (rn.Min.Y >> FPM) + r.Min.Y
	rs.Min.X = (rs.Min.X >> FPM) + r.Min.X
	rs.Min.Y = (rs.Min.Y >> FPM) + r.Min.Y

	// Round UP for Max.X and Max.Y
	if rn.Max.X&0xFF != 0 {
		rn.Max.X += 0x100
	}
	if rn.Max.Y&0xFF != 0 {
		rn.Max.Y += 0x100
	}
	if rs.Max.X&0xFF != 0 {
		rs.Max.X += 0x100
	}
	if rs.Max.Y&0xFF != 0 {
		rs.Max.Y += 0x100
	}

	rn.Max.X = (rn.Max.X >> FPM) + r.Min.X
	rn.Max.Y = (rn.Max.Y >> FPM) + r.Min.Y
	rs.Max.X = (rs.Max.X >> FPM) + r.Min.X
	rs.Max.Y = (rs.Max.Y >> FPM) + r.Min.Y

	// TODO: Implement generator for mask, based on line + bounding box
	xn, yn = xn+(r.Min.X<<FPM), yn+(r.Min.Y<<FPM)
	xs, ys = ys+(r.Min.X<<FPM), yn+(r.Min.Y<<FPM)

	dn = &dipoleMask{
		N:    dp.N,
		S:    dp.S,
		Mask: nm.Intersect(dp.Mask),
	}

	ds = &dipoleMask{
		N:    dp.N,
		S:    dp.S,
		Mask: sm.Intersect(dp.Mask),
	}

	return
}
