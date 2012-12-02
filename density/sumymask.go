package density

import (
	"image"
)

type SumYMask struct {
	Points []int
	Rect   image.Rectangle
	Mass   uint64
}

func (sym *SumYMask) Bounds() image.Rectangle {
	return sym.Rect
}

func (sym *SumYMask) MaskedMass(sy *SumY) uint64 {
	var m uint64
	for xi, yi, i := sym.Rect.Min.X, sym.Rect.Min.Y, 0; i < len(sym.Points); i += 2 {
		mi := uint64(sym.Points[i])
		yi1 := sym.Points[i+1]
		if yi1 > yi {
			m += mi * (sy.ValueAt(xi, yi1) - sy.ValueAt(xi, yi))
		} else {
			m += mi * (sy.ValueAt(xi, sym.Rect.Max.Y-1) - sy.ValueAt(xi, yi))
			xi++
		}
		yi = yi1
	}
	return m / 0xFFFF
}

func (sym *SumYMask) Wx(sy *SumY) uint64 {
	var wx uint64
	for yi, xi, i := sym.Rect.Min.Y, sym.Rect.Min.X, 0; i < len(sym.Points); i += 2 {
		m, mi := uint64(0), uint64(sym.Points[i])
		yi1 := sym.Points[i+1]
		if yi1 > yi {
			m += mi * (sy.ValueAt(xi, yi1) - sy.ValueAt(xi, yi))
		} else {
			m += mi * (sy.ValueAt(xi, sym.Rect.Max.Y-1) - sy.ValueAt(xi, yi))
			wx += uint64(xi-sym.Rect.Min.X) * m
			xi++
		}
		yi = yi1
	}
	return wx / 0xFFFF
}

func (sym *SumYMask) ToMap() Map {
	var dm, dwx, dwy uint64
	it := newIter(sym.Points)
	r := sym.Rect
	w, h := r.Dx(), r.Dy()
	dv := make([]uint16, w*h)
	for x, xi := 0, 0; x < w; x++ {
		for y := 0; y < h; y++ {
			dv[x+y*w] = uint16(it.m)
			dm += it.m
			dwx += it.m * uint64(x)
			dwy += it.m * uint64(y)
			if y+r.Min.Y > it.pt && x+r.Min.X == xi {
				if it.pti <= it.pt {
					xi++
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

func (sym0 *SumYMask) Intersect(sym *SumYMask) *SumYMask {
	// Essentially, SumYMask is just a transposed SumXMask,
	// so we cheat a bit by transposing the X and Y 
	// coordinates, converting to a SumX, re-using that 
	// type's Intersect, then transposing-and-converting
	// back to a SumYMask.
	r0 := sym0.Rect
	r0.Min.X, r0.Min.Y = r0.Min.Y, r0.Min.X
	r0.Max.X, r0.Max.Y = r0.Max.Y, r0.Max.X
	r1 := sym.Rect
	r1.Min.X, r1.Min.Y = r1.Min.Y, r1.Min.X
	r1.Max.X, r1.Max.Y = r1.Max.Y, r1.Max.X

	sxm0 := SumXMask{sym0.Points, r0, sym0.Mass}
	sxm := sxm0.Intersect(&SumXMask{sym.Points, r1, sym.Mass})

	sxm.Rect.Min.X, sxm.Rect.Min.Y = sxm.Rect.Min.Y, sxm.Rect.Min.X
	sxm.Rect.Max.X, sxm.Rect.Max.Y = sxm.Rect.Max.Y, sxm.Rect.Max.X

	return &SumYMask{sxm.Points, sxm.Rect, sxm.Mass}
}

func (sym *SumYMask) ValueAt(x, y int) (v uint64) {
	if !(image.Point{x, y}.In(sym.Rect)) {
		return
	}
	xi := sym.Rect.Min.X
	yi := sym.Rect.Min.Y
	i := 1
	// Advance to the right column
	for ; xi < x; yi, i = sym.Points[i], i+2 {
		if sym.Points[i] < yi {
			xi++
		}
	}
	// Advance to the right row.
	for yi < y && sym.Points[i] > yi {
		i += 2
	}
	return uint64(sym.Points[i-1])
}
