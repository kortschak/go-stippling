package density

import (
	"image"
)

type SumXMask struct {
	Points []int
	Rect   image.Rectangle
	Mass   uint64
}

func (sxm *SumXMask) Bounds() image.Rectangle {
	return sxm.Rect
}

func (sxm *SumXMask) ValueAt(x, y int) (v uint64) {
	if !(image.Point{x, y}.In(sxm.Rect)) {
		return
	}
	xi := sxm.Rect.Min.X
	yi := sxm.Rect.Min.Y
	i := 1
	for ; yi < y; xi, i = sxm.Points[i], i+2 {
		if sxm.Points[i] < xi {
			yi++
		}
	}
	for xi < x && sxm.Points[i] > xi {
		i += 2
	}
	return uint64(sxm.Points[i-1])
}

func (sxm *SumXMask) MaskedMass(sx *SumX) uint64 {
	if sx.Rect.Intersect(sxm.Rect).Empty() {
		return 0
	}
	var m uint64
	for xi, yi, i := sxm.Rect.Min.X, sxm.Rect.Min.Y, 0; i < len(sxm.Points); i += 2 {
		mi := uint64(sxm.Points[i])
		xi1 := sxm.Points[i+1]
		if xi1 > xi {
			m += mi * (sx.ValueAt(xi1, yi) - sx.ValueAt(xi, yi))
		} else {
			m += mi * (sx.ValueAt(sxm.Rect.Max.X-1, yi) - sx.ValueAt(xi, yi))
			yi++
		}
		xi = xi1
	}
	return m / 0xFFFF
}

func (sxm *SumXMask) Wy(sx *SumX) uint64 {
	var wy uint64
	for xi, yi, i := sxm.Rect.Min.X, sxm.Rect.Min.Y, 0; i < len(sxm.Points); i += 2 {
		m, mi := uint64(0), uint64(sxm.Points[i])
		xi1 := sxm.Points[i+1]
		if xi1 > xi {
			m += mi * (sx.ValueAt(xi1, yi) - sx.ValueAt(xi, yi))
		} else {
			m += mi * (sx.ValueAt(sxm.Rect.Max.X-1, yi) - sx.ValueAt(xi, yi))
			wy += uint64(yi-sxm.Rect.Min.Y) * m
			yi++
		}
		xi = xi1
	}
	return wy / 0xFFFF
}

func (sxm *SumXMask) ToMap() Map {
	var dm, dwx, dwy uint64
	it := newIter(sxm.Points)
	r := sxm.Rect
	w, h := r.Dx(), r.Dy()
	dv := make([]uint16, w*h)
	for y, yi := 0, 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dv[x+y*w] = uint16(it.m)
			dm += it.m
			dwx += it.m * uint64(x)
			dwy += it.m * uint64(y)
			if x+r.Min.X > it.pt && y+r.Min.Y == yi {
				if it.pti <= it.pt {
					yi++
				}
				it.next()
			}
		}
	}
	return Map{
		Values: dv,
		Stride: w,
		Rect:   r,
		mass:   dm,
		wx:     dwx,
		wy:     dwy,
	}
}

func (sxm0 *SumXMask) Intersect(sxm1 *SumXMask) *SumXMask {
	if !sxm1.Rect.In(sxm0.Rect) {
		return nil
	}

	sxm := &SumXMask{Rect: sxm1.Rect.Intersect(sxm0.Rect)}
	r := sxm.Rect
	// Since we're only saving the outline, and the most
	// common usecase is a mask without inward angles,
	// one would expect the total length of the outline
	// to be less than or equal to the outline of the
	// bounding rectangle.
	sxm.Points = make([]int, 0, 4*(sxm.Rect.Dx()+sxm.Rect.Dy()))

	// We're applying the masks to each other, and saving 
	// the result as a new mask. 

	y0 := sxm0.Rect.Min.Y
	y1 := sxm1.Rect.Min.Y

	x0it := newIter(sxm0.Points)
	x1it := newIter(sxm1.Points)
	x0it.pt = sxm0.Rect.Min.X
	x1it.pt = sxm1.Rect.Min.X

	// Move y0 and y1 to the top line of the bounding box
	for y0 < r.Min.Y {
		if x0it.pti <= x0it.pt {
			y0++
		}
		x0it.next()
	}

	for y1 < r.Min.Y {
		if x1it.pti <= x1it.pt {
			y1++
		}
		x1it.next()
	}

	// Apply both masks to each other, within
	// the bounds of their intersection.
	for y := r.Min.Y; y < r.Max.Y; y++ {

		// We have to check if x-coordinates are inside the
		// bounding rectangle, and clip the results to r.
		// Note that a mask can go to a new line while the
		// x-values are less than r.Min.X - this effectively
		// applies the current mask value to the entire line.
		for ; x0it.pti < r.Min.X; x0it.next() {
			if x0it.pti >= r.Max.X || x0it.pti <= x0it.pt {
				y0++
				break
			}
		}
		for ; x1it.pti < r.Min.X; x1it.next() {
			if x1it.pti >= r.Max.X || x1it.pti <= x1it.pt {
				y1++
				break
			}
		}

		if y0 > y && y1 > y {
			// apply both mask values to the entire line
			sxm.Points = append(sxm.Points, int((x0it.mi*x1it.mi+0x7FFF)/0xFFFF), r.Max.X-1)
			y++
		} else if y0 == y && y1 == y {
			// keep advancing x0it and x1it, using the closest point every 
			// time, until either or both have moved to a new line.
			for y0 == y && y1 == y {
				if x0it.pti == x1it.pti {
					sxm.Points = append(sxm.Points, int((x0it.mi*x1it.mi+0x7FFF)/0xFFFF), x0it.pti)
					x0it.next()
					x1it.next()
					// In this case is it is theoretically possible that
					// both x0it and x1it advanced a line simultaneously.
					if (x0it.pti >= r.Max.X || x0it.pti <= x0it.pt) &&
						(x1it.pti >= r.Max.X || x1it.pti <= x1it.pt) {
						y0++
						y1++
						sxm.Points = append(sxm.Points, int((x0it.mi*x1it.mi+0x7FFF)/0xFFFF), r.Max.X-1)
					} else {
						if x0it.pti >= r.Max.X || x0it.pti <= x0it.pt {
							y0++
						}
						if x1it.pti >= r.Max.X || x1it.pti <= x1it.pt {
							y1++
						}
					}
				} else if x0it.pti < x1it.pti {
					sxm.Points = append(sxm.Points, int((x0it.mi*x1it.mi+0x7FFF)/0xFFFF), x0it.pti)
					x0it.next()
					if x0it.pti >= r.Max.X || x0it.pti <= x0it.pt {
						y0++
					}
				} else {
					sxm.Points = append(sxm.Points, int((x0it.mi*x1it.mi+0x7FFF)/0xFFFF), x1it.pti)
					x1it.next()
					if x1it.pti >= r.Max.X || x1it.pti <= x1it.pt {
						y1++
					}
				}
			}
		}

		if y0 == y && y1 > y {
			// keep advancing x0it until it reaches a new line
			for x0it.pti < r.Max.X && x0it.pti > x0it.pt {
				sxm.Points = append(sxm.Points, int((x0it.mi*x1it.m+0x7FFF)/0xFFFF), x0it.pti)
				x0it.next()
			}
			sxm.Points = append(sxm.Points, int((x0it.mi*x1it.m+0x7FFF)/0xFFFF), r.Max.X-1)
			x0it.next()
			y0++
		} else if y1 > y && y1 == y {
			// keep advancing x1it until it reaches a new line
			for x1it.pti < r.Max.X && x1it.pti > x1it.pt {
				sxm.Points = append(sxm.Points, int((x0it.m*x1it.mi+0x7FFF)/0xFFFF), x1it.pti)
				x1it.next()
			}
			sxm.Points = append(sxm.Points, int((x0it.m*x1it.mi+0x7FFF)/0xFFFF), r.Max.X-1)
			x1it.next()
			y1++
		}
	}

	// Prune the repeated mask values 
	for i := 0; i < len(sxm.Points)-2; i += 2 {
		if sxm.Points[i] == sxm.Points[i+2] && sxm.Points[i+1] < sxm.Points[i+3] {
			sxm.Points = append(sxm.Points[:i], sxm.Points[(i+2):]...)
		}
	}

	// Prune "wrap around line" mask values
	for i := 0; i < len(sxm.Points)-2; i += 2 {
		if sxm.Points[i] == sxm.Points[i+2] && sxm.Points[i+3] < sxm.Points[i+1] {
			sxm.Points = append(sxm.Points[:i], sxm.Points[(i+2):]...)
		}
	}

	// Add "terminating" point (for iterator)
	sxm.Points = append(sxm.Points, 0, r.Min.X-1)

	// Calculate mass
	for x, i := r.Min.X, 0; i < len(sxm.Points); i += 2 {
		if sxm.Points[i+1] <= x {
			sxm.Mass += uint64((r.Max.X - x) * sxm.Points[i])
		} else {
			sxm.Mass += uint64((sxm.Points[i+1] - x) * sxm.Points[i])
		}
		x = sxm.Points[i+1]
	}

	return sxm
}
