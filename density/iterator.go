package density

// Mask iterator. Used for... well.. iterating over
// the values of a mask, and storing the current
// and next values in it.
type MaskIter struct {
	pts []int
	pt  int
	pti int
	i   int
	m   uint64
	mi  uint64
}

// Note that this function doesn't do any bounds checking
func (m *MaskIter) next() {
	m.i += 2
	m.m = m.mi
	m.pt = m.pti
	m.mi = uint64(m.pts[m.i])
	m.pti = m.pts[m.i+1]
}

func newIter(points []int) (it MaskIter) {
	if len(points) > 1 {
		it.m = uint64(points[0])
		it.pti = points[1]
		it.pts = points
	}
	return
}
