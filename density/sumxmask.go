package density

import (
	"image"
)

type SumXMask struct {
	Points [][]int
	Rect   image.Rectangle
	// Mass of last masked SumX
	Mass float64
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
			dv = line[i+1]
		}
		v = dv
	}
	return
}

func (sxm *SumXMask) ApplyTo(sx *SumX) {
	if !sx.Rect.Intersect(sxm.Rect).Empty() {
		var mass, wy uint64
		for y := 0; y < sxm.Rect.Dy(); y++ {
			line := sxm.Points[y]
			var linemass, pv uint64
			for x := 0; x < sxm.Rect.Dx(); x++ {
				v := sx.ValueAt(line[x*2], y+sxm.Rect.Min.Y)
				linemass += (v - pv) * line[x*2+1]
				pv = v
			}
			mass += linemass
			wy += linemass * y
		}
		sxm.Mass = float64(mass) / float64(sxm.Range)
		sxm.Wy = float64(wy)/float64(mass) + float64(sxm.Rect.Min.Y)
	}
	return
}
