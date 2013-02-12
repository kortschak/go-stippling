package density

import (
	"image"
)

type SumYMask struct {
	Points [][]int
	Rect   image.Rectangle
	// Mass of last masked SumY
	Mass float64
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
		var mass, wx uint64
		for x := 0; x < sym.Rect.Dx(); x++ {
			column := sym.Points[x]
			var columnmass, pv uint64
			for y := 0; y < sym.Rect.Dy(); y++ {
				v := sy.ValueAt(x+sym.Rect.Min.X, column[y*2])
				columnmass += (v - pv) * uint64(column[y*2+1])
				pv = v
			}
			mass += columnmass
			wx += columnmass * uint64(x)
		}
		sym.Mass = float64(mass) / float64(sym.Range)
		sym.Wx = float64(wx)/float64(mass) + float64(sym.Rect.Min.X)
	}
	return
}
