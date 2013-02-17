package density

import (
	"image"
)

type SumYMask struct {
	// Mask value from last point up to this point
	Points [][]int
	Rect   image.Rectangle
	// Mass of last masked SumY, and the mass of SumY
	Mass, MaskMass float64
	// Weighed average X value of last masked SumY
	Wx float64
	// Range of mask value
	Range int
}

func (sym *SumYMask) Bounds() image.Rectangle {
	return sym.Rect
}

func (sym *SumYMask) ValueAt(x, y int) (v uint64) {
	if (image.Point{x, y}.In(sym.Rect)) {
		column := sym.Points[y-sym.Rect.Min.Y]
		var dv uint64
		for i := 0; i < len(column) && column[i] < x; i += 2 {
			dv = uint64(column[i+1])
		}
		v = dv
	}
	return
}

func (sym *SumYMask) ApplyTo(sy *SumY) {
	if !sy.Rect.Intersect(sym.Rect).Empty() {
		var mass, maskmass, wx uint64
		for x, column := range sym.Points {
			var columnmass, pv uint64
			py := sym.Rect.Min.Y
			for i := 0; i < sym.Rect.Dy(); i++ {
				y := column[i*2] - 1
				mask := uint64(column[i*2+1])
				v := sy.ValueAt(x+sym.Rect.Min.X, y)
				columnmass += (v - pv) * mask
				pv = v
				maskmass += mask * uint64(y-py)
				py = y
			}
			mass += columnmass
			wx += columnmass * uint64(x)
		}
		sym.Mass = float64(mass) / float64(sym.Range)
		sym.MaskMass = float64(maskmass)
		sym.Wx = float64(wx)/float64(mass) + float64(sym.Rect.Min.X)
	}
	return
}
