package density

import (
	"image"
	"image/color"
)

/*
CubeSum is similar to DSum, but extends the functionality to
three axes: x, y and z. It allows for comparing accross
frames of a movie (which can be appended frame by frame).
Being a volume it obviously consumes a lot of memory, so use
with care. Also note that there is a risk of overflowing the
precomputed values if too many frames are stacked in one cube.
*/
type CubeSum struct {
	// Values holds the map's density values. The value at (x, y)
	// starts at Values[z*Rect.Dx()*Rect.Dy() + (y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*1].
	Values []uint64
	// Stride is the Values' stride between
	// vertically adjacent pixels.
	Stride int
	// Rect is the Map's bounds.
	Rect image.Rectangle
	// LenZ is the current number of frames, CapZ is the total capacity
	LenZ, CapZ int
}

func (cbs *CubeSum) ColorModel() color.Model {
	return color.Gray16Model
}

func (cbs *CubeSum) Copy(s *CubeSum) {
	cbs.Values = make([]uint64, len(s.Values), cap(s.Values))
	copy(cbs.Values, s.Values)
	cbs.Stride = s.Stride
	cbs.Rect = s.Rect
	cbs.LenZ = s.LenZ
	cbs.CapZ = s.CapZ

}

func (cbs *CubeSum) DVOffSet(x, y, z int) int {
	return z*cbs.Rect.Dx()*cbs.Rect.Dy() + (y-cbs.Rect.Min.Y)*cbs.Stride + (x - cbs.Rect.Min.X)
}

func (cbs *CubeSum) Bounds() image.Rectangle { return cbs.Rect }

// Shows the last added frame
func (cbs *CubeSum) At(x, y int) (v color.Color) {
	x1y1z1 := cbs.ValueAt(x, y, cbs.LenZ-1)
	x0y0z1 := cbs.ValueAt(x-1, y-1, cbs.LenZ-1)
	x0y1z1 := cbs.ValueAt(x-1, y, cbs.LenZ-1)
	x1y0z1 := cbs.ValueAt(x, y-1, cbs.LenZ-1)
	x1y1z0 := cbs.ValueAt(x, y, cbs.LenZ-2)
	x0y0z0 := cbs.ValueAt(x-1, y-1, cbs.LenZ-2)
	x0y1z0 := cbs.ValueAt(x-1, y, cbs.LenZ-2)
	x1y0z0 := cbs.ValueAt(x, y-1, cbs.LenZ-2)
	v = color.Gray16{uint16((x1y1z1 - x0y1z1 - x1y0z1 + x0y0z1) - (x1y1z0 - x0y1z0 - x1y0z0 + x0y0z0))}
	return
}

func (cbs *CubeSum) ValueAt(x, y, z int) (v uint64) {
	if (image.Point{x, y}.In(cbs.Rect)) && z >= 0 && z < cbs.LenZ {
		i := cbs.DVOffSet(x, y, z)
		v = cbs.Values[i]
	}
	return
}

func (cbs *CubeSum) NegValueAt(x, y, z int) (v uint64) {
	if (image.Point{x, y}.In(cbs.Rect)) && z >= 0 && z < cbs.LenZ {
		i := cbs.DVOffSet(x, y, z)
		v = uint64(x+1-cbs.Rect.Min.X)*uint64(y+1-cbs.Rect.Min.Y)*uint64(z+1)*0xFFFF - cbs.Values[i]
	}
	return
}

// Sums the volume defined by the rectangle and zmin-zmax. Inclusive min, exclusive max (like image.Rectangle)
func (cbs *CubeSum) VolumeSum(r image.Rectangle, zmin, zmax int) uint64 {
	r = r.Intersect(cbs.Rect).Sub(image.Point{1, 1})
	if zmax >= cbs.LenZ {
		zmax = cbs.LenZ - 1
	} else {
		zmax -= 1
	}
	zmin--

	return cbs.ValueAt(r.Max.X, r.Max.Y, zmax) + cbs.ValueAt(r.Min.X, r.Min.Y, zmax) -
		cbs.ValueAt(r.Min.X, r.Max.Y, zmax) - cbs.ValueAt(r.Max.X, r.Min.Y, zmax) -
		(cbs.ValueAt(r.Max.X, r.Max.Y, zmin) + cbs.ValueAt(r.Min.X, r.Min.Y, zmin) -
			cbs.ValueAt(r.Min.X, r.Max.Y, zmin) - cbs.ValueAt(r.Max.X, r.Min.Y, zmin))
}

// Like Sum, but gives the value of (volume*0xFFFF - Sum) - the negative space, essentially
func (cbs *CubeSum) NegVolumeSum(r image.Rectangle, zmin, zmax int) uint64 {
	r = r.Intersect(cbs.Rect).Sub(image.Point{1, 1})
	if zmax >= cbs.LenZ {
		zmax = cbs.LenZ - 1
	} else {
		zmax -= 1
	}
	zmin--
	volumeMass := uint64(r.Dx()) * uint64(r.Dy()) * uint64(zmax-zmin) * 0xFFFF
	return volumeMass -
		cbs.ValueAt(r.Max.X, r.Max.Y, zmax) - cbs.ValueAt(r.Min.X, r.Min.Y, zmax) +
		cbs.ValueAt(r.Min.X, r.Max.Y, zmax) + cbs.ValueAt(r.Max.X, r.Min.Y, zmax) +
		cbs.ValueAt(r.Max.X, r.Max.Y, zmin) + cbs.ValueAt(r.Min.X, r.Min.Y, zmin) -
		cbs.ValueAt(r.Min.X, r.Max.Y, zmin) - cbs.ValueAt(r.Max.X, r.Min.Y, zmin)
}

// Given a Rectangle and zmin/zmax, finds x closest to line dividing
// the mass of the cube bound by these coordinates in half.
func (cbs *CubeSum) FindCx(r image.Rectangle, zmin, zmax int) int {
	xmin := r.Min.X
	xmax := r.Max.X
	r = r.Sub(image.Point{1, 1})
	zmin--
	zmax--
	x := (xmax + xmin + 1) / 2
	xmaxymaxzmax := cbs.ValueAt(r.Max.X, r.Max.Y, zmax)
	xminyminzmax := cbs.ValueAt(r.Min.X, r.Min.Y, zmax)
	xminymaxzmax := cbs.ValueAt(r.Min.X, r.Max.Y, zmax)
	xmaxyminzmax := cbs.ValueAt(r.Max.X, r.Min.Y, zmax)
	xmaxymaxzmin := cbs.ValueAt(r.Max.X, r.Max.Y, zmin)
	xminyminzmin := cbs.ValueAt(r.Min.X, r.Min.Y, zmin)
	xminymaxzmin := cbs.ValueAt(r.Min.X, r.Max.Y, zmin)
	xmaxyminzmin := cbs.ValueAt(r.Max.X, r.Min.Y, zmin)
	for {
		// The centre of mass is probably not a round number,
		// so we aim to iterate only to the margin of 1 pixel
		if xmax-xmin > 1 {
			cxyminzmax := cbs.ValueAt(x, r.Min.Y, zmax)
			cxymaxzmax := cbs.ValueAt(x, r.Max.Y, zmax)
			cxyminzmin := cbs.ValueAt(x, r.Min.Y, zmin)
			cxymaxzmin := cbs.ValueAt(x, r.Max.Y, zmin)
			lmass := (cxymaxzmax - cxyminzmax - xminymaxzmax + xminyminzmax) -
				(cxymaxzmin - cxyminzmin - xminymaxzmin + xminyminzmin)
			rmass := (xmaxymaxzmax - cxymaxzmax - xmaxyminzmax + cxyminzmax) -
				(xmaxymaxzmin - cxymaxzmin - xmaxyminzmin + cxyminzmin)
			if lmass < rmass {
				xmin = x
				x = (x + xmax + 1) / 2
			} else {
				xmax = x
				x = (x + xmin + 1) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Note that lmass and rmass are guaranteed to be smaller than total mass
			cxyminzmax := cbs.ValueAt(xmin, r.Min.Y, zmax)
			cxymaxzmax := cbs.ValueAt(xmin, r.Max.Y, zmax)
			cxyminzmin := cbs.ValueAt(xmin, r.Min.Y, zmin)
			cxymaxzmin := cbs.ValueAt(xmin, r.Max.Y, zmin)
			lmass := (cxymaxzmax - cxyminzmax - xminymaxzmax + xminyminzmax) -
				(cxymaxzmin - cxyminzmin - xminymaxzmin + xminyminzmin)
			cxyminzmax = cbs.ValueAt(xmax, r.Min.Y, zmax)
			cxymaxzmax = cbs.ValueAt(xmax, r.Max.Y, zmax)
			cxyminzmin = cbs.ValueAt(xmax, r.Min.Y, zmin)
			cxymaxzmin = cbs.ValueAt(xmax, r.Max.Y, zmin)
			rmass := (xmaxymaxzmax - cxymaxzmax - xmaxyminzmax + cxyminzmax) -
				(xmaxymaxzmin - cxymaxzmin - xmaxyminzmin + cxyminzmin)
				//			tmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				//				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			if lmass > rmass {
				x = xmin
			} else {
				x = xmax
			}
			break
		}
	}
	return x
}

// Given a Rectangle and zmin/zmax, finds y closest to line dividing
// the mass of the cube bound by these coordinates in half.
func (cbs *CubeSum) FindCy(r image.Rectangle, zmin, zmax int) int {
	ymin := r.Min.Y
	ymax := r.Max.Y
	r = r.Sub(image.Point{1, 1})
	zmax--
	zmin--
	y := (ymax + ymin + 1) / 2
	xmaxymaxzmax := cbs.ValueAt(r.Max.X, r.Max.Y, zmax)
	xminyminzmax := cbs.ValueAt(r.Min.X, r.Min.Y, zmax)
	xminymaxzmax := cbs.ValueAt(r.Min.X, r.Max.Y, zmax)
	xmaxyminzmax := cbs.ValueAt(r.Max.X, r.Min.Y, zmax)
	xmaxymaxzmin := cbs.ValueAt(r.Max.X, r.Max.Y, zmin)
	xminyminzmin := cbs.ValueAt(r.Min.X, r.Min.Y, zmin)
	xminymaxzmin := cbs.ValueAt(r.Min.X, r.Max.Y, zmin)
	xmaxyminzmin := cbs.ValueAt(r.Max.X, r.Min.Y, zmin)
	for {
		if ymax-ymin > 1 {
			xmincyzmax := cbs.ValueAt(r.Min.X, y, zmax)
			xmaxcyzmax := cbs.ValueAt(r.Max.X, y, zmax)
			xmincyzmin := cbs.ValueAt(r.Min.X, y, zmin)
			xmaxcyzmin := cbs.ValueAt(r.Max.X, y, zmin)
			upmass := (xmaxcyzmax - xmincyzmax - xmaxyminzmax + xminyminzmax) -
				(xmaxcyzmin - xmincyzmin - xmaxyminzmin + xminyminzmin)
			downmass := (xmaxymaxzmax - xmaxcyzmax - xminymaxzmax + xmincyzmax) -
				(xmaxymaxzmin - xmaxcyzmin - xminymaxzmin + xmincyzmin)
			if upmass < downmass {
				ymin = y
				y = (y + ymax + 1) / 2
			} else {
				ymax = y
				y = (y + ymin + 1) / 2
			}
		} else {
			xmincyzmax := cbs.ValueAt(r.Min.X, ymin, zmax)
			xmaxcyzmax := cbs.ValueAt(r.Max.X, ymin, zmax)
			xmincyzmin := cbs.ValueAt(r.Min.X, ymin, zmin)
			xmaxcyzmin := cbs.ValueAt(r.Max.X, ymin, zmin)
			upmass := (xmaxcyzmax - xmincyzmax - xmaxyminzmax + xminyminzmax) -
				(xmaxcyzmin - xmincyzmin - xmaxyminzmin + xminyminzmin)
			xmincyzmax = cbs.ValueAt(r.Min.X, ymax, zmax)
			xmaxcyzmax = cbs.ValueAt(r.Max.X, ymax, zmax)
			xmincyzmin = cbs.ValueAt(r.Min.X, ymax, zmin)
			xmaxcyzmin = cbs.ValueAt(r.Max.X, ymax, zmin)
			downmass := (xmaxymaxzmax - xmaxcyzmax - xminymaxzmax + xmincyzmax) -
				(xmaxymaxzmin - xmaxcyzmin - xminymaxzmin + xmincyzmin)
				//			tmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				//				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			if upmass > downmass {
				y = ymin
			} else {
				y = ymax
			}
			break
		}
	}
	return y
}

// Given a Rectangle and zmin/zmax, finds y closest to line dividing
// the mass of the cube bound by these coordinates in half.
func (cbs *CubeSum) FindCz(r image.Rectangle, MinZ, MaxZ int) int {
	zmin := MinZ
	zmax := MaxZ
	r = r.Sub(image.Point{1, 1})
	MinZ--
	MaxZ--
	z := (zmax + zmin + 1) / 2
	xmaxymaxzmax := cbs.ValueAt(r.Max.X, r.Max.Y, MaxZ)
	xminyminzmax := cbs.ValueAt(r.Min.X, r.Min.Y, MaxZ)
	xminymaxzmax := cbs.ValueAt(r.Min.X, r.Max.Y, MaxZ)
	xmaxyminzmax := cbs.ValueAt(r.Max.X, r.Min.Y, MaxZ)
	xmaxymaxzmin := cbs.ValueAt(r.Max.X, r.Max.Y, MinZ)
	xminyminzmin := cbs.ValueAt(r.Min.X, r.Min.Y, MinZ)
	xminymaxzmin := cbs.ValueAt(r.Min.X, r.Max.Y, MinZ)
	xmaxyminzmin := cbs.ValueAt(r.Max.X, r.Min.Y, MinZ)
	for {
		if zmax-zmin > 1 {
			xmaxymaxcz := cbs.ValueAt(r.Max.X, r.Max.Y, z)
			xminymaxcz := cbs.ValueAt(r.Min.X, r.Max.Y, z)
			xmaxymincz := cbs.ValueAt(r.Max.X, r.Min.Y, z)
			xminymincz := cbs.ValueAt(r.Min.X, r.Min.Y, z)
			frontmass := (xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz) -
				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			backmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				(xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz)
			if frontmass < backmass {
				zmin = z
				z = (z + zmax + 1) / 2
			} else {
				zmax = z
				z = (z + zmin + 1) / 2
			}
		} else {
			xmaxymaxcz := cbs.ValueAt(r.Max.X, r.Max.Y, zmin)
			xminymaxcz := cbs.ValueAt(r.Min.X, r.Max.Y, zmin)
			xmaxymincz := cbs.ValueAt(r.Max.X, r.Min.Y, zmin)
			xminymincz := cbs.ValueAt(r.Min.X, r.Min.Y, zmin)
			frontmass := (xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz) -
				(xmaxymaxzmin - xmaxyminzmin - xminyminzmin + xminyminzmin)
			xmaxymaxcz = cbs.ValueAt(r.Max.X, r.Max.Y, zmax)
			xminymaxcz = cbs.ValueAt(r.Min.X, r.Max.Y, zmax)
			xmaxymincz = cbs.ValueAt(r.Max.X, r.Min.Y, zmax)
			xminymincz = cbs.ValueAt(r.Min.X, r.Min.Y, zmax)
			backmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				(xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz)
				//			tmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				//				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			if frontmass > backmass {
				z = zmin
			} else {
				z = zmax
			}
			break
		}
	}
	return z
}

// Given a Rectangle and zmin/zmax, finds x closest to line dividing
// the "negative" mass of the cube bound by these coordinates mass in half.
func (cbs *CubeSum) FindNegCx(r image.Rectangle, zmin, zmax int) int {
	xmin := r.Min.X
	xmax := r.Max.X
	r = r.Sub(image.Point{1, 1})
	zmin--
	zmax--
	x := (xmax + xmin + 1) / 2
	xmaxymaxzmax := cbs.NegValueAt(r.Max.X, r.Max.Y, zmax)
	xminyminzmax := cbs.NegValueAt(r.Min.X, r.Min.Y, zmax)
	xminymaxzmax := cbs.NegValueAt(r.Min.X, r.Max.Y, zmax)
	xmaxyminzmax := cbs.NegValueAt(r.Max.X, r.Min.Y, zmax)
	xmaxymaxzmin := cbs.NegValueAt(r.Max.X, r.Max.Y, zmin)
	xminyminzmin := cbs.NegValueAt(r.Min.X, r.Min.Y, zmin)
	xminymaxzmin := cbs.NegValueAt(r.Min.X, r.Max.Y, zmin)
	xmaxyminzmin := cbs.NegValueAt(r.Max.X, r.Min.Y, zmin)
	for {
		// The centre of mass is probably not a round number,
		// so we aim to iterate only to the margin of 1 pixel
		if xmax-xmin > 1 {
			cxyminzmax := cbs.NegValueAt(x, r.Min.Y, zmax)
			cxymaxzmax := cbs.NegValueAt(x, r.Max.Y, zmax)
			cxyminzmin := cbs.NegValueAt(x, r.Min.Y, zmin)
			cxymaxzmin := cbs.NegValueAt(x, r.Max.Y, zmin)
			lmass := (cxymaxzmax - cxyminzmax - xminymaxzmax + xminyminzmax) -
				(cxymaxzmin - cxyminzmin - xminymaxzmin + xminyminzmin)
			rmass := (xmaxymaxzmax - cxymaxzmax - xmaxyminzmax + cxyminzmax) -
				(xmaxymaxzmin - cxymaxzmin - xmaxyminzmin + cxyminzmin)
			if lmass < rmass {
				xmin = x
				x = (x + xmax + 1) / 2
			} else {
				xmax = x
				x = (x + xmin + 1) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Note that lmass and rmass are guaranteed to be smaller than total mass
			cxyminzmax := cbs.NegValueAt(xmin, r.Min.Y, zmax)
			cxymaxzmax := cbs.NegValueAt(xmin, r.Max.Y, zmax)
			cxyminzmin := cbs.NegValueAt(xmin, r.Min.Y, zmin)
			cxymaxzmin := cbs.NegValueAt(xmin, r.Max.Y, zmin)
			lmass := (cxymaxzmax - cxyminzmax - xminymaxzmax + xminyminzmax) -
				(cxymaxzmin - cxyminzmin - xminymaxzmin + xminyminzmin)
			cxyminzmax = cbs.NegValueAt(xmax, r.Min.Y, zmax)
			cxymaxzmax = cbs.NegValueAt(xmax, r.Max.Y, zmax)
			cxyminzmin = cbs.NegValueAt(xmax, r.Min.Y, zmin)
			cxymaxzmin = cbs.NegValueAt(xmax, r.Max.Y, zmin)
			rmass := (xmaxymaxzmax - cxymaxzmax - xmaxyminzmax + cxyminzmax) -
				(xmaxymaxzmin - cxymaxzmin - xmaxyminzmin + cxyminzmin)
				//			tmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				//				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			if lmass > rmass {
				x = xmin
			} else {
				x = xmax
			}
			break
		}
	}
	return x
}

// Given a Rectangle and zmin/zmax, finds y closest to line dividing
// the "negative" mass of the cube bound by these coordinates mass in half.
func (cbs *CubeSum) FindNegCy(r image.Rectangle, zmin, zmax int) int {
	ymin := r.Min.Y
	ymax := r.Max.Y
	r = r.Sub(image.Point{1, 1})
	zmax--
	zmin--
	y := (ymax + ymin + 1) / 2
	xmaxymaxzmax := cbs.NegValueAt(r.Max.X, r.Max.Y, zmax)
	xminyminzmax := cbs.NegValueAt(r.Min.X, r.Min.Y, zmax)
	xminymaxzmax := cbs.NegValueAt(r.Min.X, r.Max.Y, zmax)
	xmaxyminzmax := cbs.NegValueAt(r.Max.X, r.Min.Y, zmax)
	xmaxymaxzmin := cbs.NegValueAt(r.Max.X, r.Max.Y, zmin)
	xminyminzmin := cbs.NegValueAt(r.Min.X, r.Min.Y, zmin)
	xminymaxzmin := cbs.NegValueAt(r.Min.X, r.Max.Y, zmin)
	xmaxyminzmin := cbs.NegValueAt(r.Max.X, r.Min.Y, zmin)
	for {
		if ymax-ymin > 1 {
			xmincyzmax := cbs.NegValueAt(r.Min.X, y, zmax)
			xmaxcyzmax := cbs.NegValueAt(r.Max.X, y, zmax)
			xmincyzmin := cbs.NegValueAt(r.Min.X, y, zmin)
			xmaxcyzmin := cbs.NegValueAt(r.Max.X, y, zmin)
			upmass := (xmaxcyzmax - xmincyzmax - xmaxyminzmax + xminyminzmax) -
				(xmaxcyzmin - xmincyzmin - xmaxyminzmin + xminyminzmin)
			downmass := (xmaxymaxzmax - xmaxcyzmax - xminymaxzmax + xmincyzmax) -
				(xmaxymaxzmin - xmaxcyzmin - xminymaxzmin + xmincyzmin)
			if upmass < downmass {
				ymin = y
				y = (y + ymax + 1) / 2
			} else {
				ymax = y
				y = (y + ymin + 1) / 2
			}
		} else {
			xmincyzmax := cbs.NegValueAt(r.Min.X, ymin, zmax)
			xmaxcyzmax := cbs.NegValueAt(r.Max.X, ymin, zmax)
			xmincyzmin := cbs.NegValueAt(r.Min.X, ymin, zmin)
			xmaxcyzmin := cbs.NegValueAt(r.Max.X, ymin, zmin)
			upmass := (xmaxcyzmax - xmincyzmax - xmaxyminzmax + xminyminzmax) -
				(xmaxcyzmin - xmincyzmin - xmaxyminzmin + xminyminzmin)
			xmincyzmax = cbs.NegValueAt(r.Min.X, ymax, zmax)
			xmaxcyzmax = cbs.NegValueAt(r.Max.X, ymax, zmax)
			xmincyzmin = cbs.NegValueAt(r.Min.X, ymax, zmin)
			xmaxcyzmin = cbs.NegValueAt(r.Max.X, ymax, zmin)
			downmass := (xmaxymaxzmax - xmaxcyzmax - xminymaxzmax + xmincyzmax) -
				(xmaxymaxzmin - xmaxcyzmin - xminymaxzmin + xmincyzmin)
				//			tmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				//				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			if upmass > downmass {
				y = ymin
			} else {
				y = ymax
			}
			break
		}
	}
	return y
}

// Given a Rectangle and zmin/zmax, finds y closest to line dividing
// the "negative" mass of the cube bound by these coordinates mass in half.
func (cbs *CubeSum) FindNegCz(r image.Rectangle, MinZ, MaxZ int) int {
	zmin := MinZ
	zmax := MaxZ
	r = r.Sub(image.Point{1, 1})
	MinZ--
	MaxZ--
	z := (zmax + zmin + 1) / 2
	xmaxymaxzmax := cbs.NegValueAt(r.Max.X, r.Max.Y, MaxZ)
	xminyminzmax := cbs.NegValueAt(r.Min.X, r.Min.Y, MaxZ)
	xminymaxzmax := cbs.NegValueAt(r.Min.X, r.Max.Y, MaxZ)
	xmaxyminzmax := cbs.NegValueAt(r.Max.X, r.Min.Y, MaxZ)
	xmaxymaxzmin := cbs.NegValueAt(r.Max.X, r.Max.Y, MinZ)
	xminyminzmin := cbs.NegValueAt(r.Min.X, r.Min.Y, MinZ)
	xminymaxzmin := cbs.NegValueAt(r.Min.X, r.Max.Y, MinZ)
	xmaxyminzmin := cbs.NegValueAt(r.Max.X, r.Min.Y, MinZ)
	for {
		if zmax-zmin > 1 {
			xmaxymaxcz := cbs.NegValueAt(r.Max.X, r.Max.Y, z)
			xminymaxcz := cbs.NegValueAt(r.Min.X, r.Max.Y, z)
			xmaxymincz := cbs.NegValueAt(r.Max.X, r.Min.Y, z)
			xminymincz := cbs.NegValueAt(r.Min.X, r.Min.Y, z)
			frontmass := (xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz) -
				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			backmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				(xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz)
			if frontmass < backmass {
				zmin = z
				z = (z + zmax + 1) / 2
			} else {
				zmax = z
				z = (z + zmin + 1) / 2
			}
		} else {
			xmaxymaxcz := cbs.NegValueAt(r.Max.X, r.Max.Y, zmin)
			xminymaxcz := cbs.NegValueAt(r.Min.X, r.Max.Y, zmin)
			xmaxymincz := cbs.NegValueAt(r.Max.X, r.Min.Y, zmin)
			xminymincz := cbs.NegValueAt(r.Min.X, r.Min.Y, zmin)
			frontmass := (xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz) -
				(xmaxymaxzmin - xmaxyminzmin - xminyminzmin + xminyminzmin)
			xmaxymaxcz = cbs.NegValueAt(r.Max.X, r.Max.Y, zmax)
			xminymaxcz = cbs.NegValueAt(r.Min.X, r.Max.Y, zmax)
			xmaxymincz = cbs.NegValueAt(r.Max.X, r.Min.Y, zmax)
			xminymincz = cbs.NegValueAt(r.Min.X, r.Min.Y, zmax)
			backmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				(xmaxymaxcz - xminymaxcz - xmaxymincz + xminymincz)
				//			tmass := (xmaxymaxzmax - xmaxyminzmax - xminymaxzmax + xminyminzmax) -
				//				(xmaxymaxzmin - xmaxyminzmin - xminymaxzmin + xminyminzmin)
			if frontmass > backmass {
				z = zmin
			} else {
				z = zmax
			}
			break
		}
	}
	return z
}

func NewCubeSum(r image.Rectangle, capz int) *CubeSum {
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h*capz)
	return &CubeSum{Values: dv, Stride: w, Rect: r, LenZ: 0, CapZ: capz}
}

func CubeSumFrom(i *image.Image, d Model, capz int) *CubeSum {
	r := (*i).Bounds()
	w, h := r.Dx(), r.Dy()
	dv := make([]uint64, w*h*capz)

	for x, vx := 0, uint64(0); x < w; x++ {
		vx += uint64(d.Convert((*i).At(x+r.Min.X, r.Min.Y)))
		dv[x] = vx
	}

	for y := 1; y < h; y++ {
		for x, vx := 0, uint64(0); x < w; x++ {
			vx += uint64(d.Convert(((*i).At(x+r.Min.X, y+r.Min.Y))))
			dv[x+y*w] = vx + dv[x+(y-1)*w]
		}
	}

	return &CubeSum{Values: dv, Stride: w, Rect: r, LenZ: 1, CapZ: capz}
}

func (cbs *CubeSum) AddFrame(i *image.Image, d Model) {
	// Only add the part that overlaps
	r := (*i).Bounds().Intersect(cbs.Rect)
	if !r.Empty() && cbs.LenZ < cbs.CapZ {
		w := r.Dx()
		h := r.Dy()
		StrideZ := w * h

		// Top row: only sum previous x
		for x, vx := 0, uint64(0); x < w; x++ {
			vx += uint64(d.Convert((*i).At(x+r.Min.X, r.Min.Y)))
			cbs.Values[x+cbs.LenZ*StrideZ] = vx
		}

		// Rest: sum previous x, then add previous y.
		for y := 1; y < h; y++ {
			for x, vx := 0, uint64(0); x < w; x++ {
				vx += uint64(d.Convert(((*i).At(x+r.Min.X, y+r.Min.Y))))
				cbs.Values[x+y*w+cbs.LenZ*StrideZ] = vx + cbs.Values[x+(y-1)*w+cbs.LenZ*StrideZ]
			}
		}
		if cbs.LenZ > 0 {
			// Now add previous z
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					cbs.Values[x+y*w+cbs.LenZ*StrideZ] += cbs.Values[x+y*w+(cbs.LenZ-1)*StrideZ]
				}
			}
		}
		cbs.LenZ++
	}
}
