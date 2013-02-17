package density

import (
	"image"
)

type SumXMask struct {
	// Mask value from last point up to this point
	Points [][]int
	Rect   image.Rectangle
	// Mass of last masked SumX, and the mass of SumX
	Mass, MaskMass float64
	// Weighed average Y value of last masked SumX
	Wy float64
	// Range of mask value
	Range int
}

func (sxm *SumXMask) Bounds() image.Rectangle {
	return sxm.Rect
}

func (sxm *SumXMask) ValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(sxm.Rect)) {
		line := sxm.Points[x-sxm.Rect.Min.X]
		var dv uint64
		for i := 0; i < len(line) && line[i] < y; i += 2 {
			dv = uint64(line[i+1])
		}
		v = dv
	}
	return
}

func (sxm *SumXMask) ApplyTo(sx *SumX) {
	if !sx.Rect.Intersect(sxm.Rect).Empty() {
		var mass, maskmass, wy uint64
		for y, line := range sxm.Points {
			var linemass, pv uint64
			px := sxm.Rect.Min.X
			for i := 0; i < len(line); i += 2 {
				x := line[i] - 1
				mask := uint64(line[i+1])
				v := sx.ValueAt(x, y+sxm.Rect.Min.Y)
				linemass += (v - pv) * mask
				pv = v
				maskmass += mask * uint64(x-px)
				px = x
			}
			mass += linemass
			wy += linemass * uint64(y)
		}
		sxm.Mass = float64(mass) / float64(sxm.Range)
		sxm.MaskMass = float64(maskmass)
		sxm.Wy = float64(wy)/float64(mass) + float64(sxm.Rect.Min.Y)
	}
	return
}
