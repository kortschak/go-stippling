package density

type SumMask struct {
	X SumXMask
	Y SumYMask
}

// Check if the masks in the X and Y fields are equal.
// Slow, use for debugging constructors and the like.
func (sm *SumMask) Valid() bool {
	if !sm.X.Rect.Eq(sm.Y.Rect) {
		return false
	}

	xmap := sm.X.ToMap()
	ymap := sm.Y.ToMap()

	for i := 0; i < len(xmap.Values); i++ {
		if xmap.Values[i] != ymap.Values[i] {
			return false
		}
	}

	return true
}

func (sm0 *SumMask) Intersect(sm *SumMask) *SumMask {
	sxm := sm0.X.Intersect(&sm.X)
	sym := sm0.Y.Intersect(&sm.Y)
	return &SumMask{*sxm, *sym}
}
