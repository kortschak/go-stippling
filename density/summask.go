package density

import (
	"image"
)

type SumMask struct {
	X SumXMask
	Y SumYMask
}

func (sm *SumMask) ApplyTo(s *Sum) {
	sm.X.ApplyTo(&s.X)
	sm.Y.ApplyTo(&s.Y)
}

func (sm *SumMask) Mass() float64 {
	return sm.X.Mass
}

func (sm *SumMask) WX() float64 {
	return sm.Y.Wx
}

func (sm *SumMask) WY() float64 {
	return sm.X.Wy
}

// Returns a new opaque SumMask
func NewSumMask(r image.Rectangle, Range int) *SumMask {
	s := new(SumMask)
	s.X.Range = Range
	s.Y.Range = Range
	s.X.Rect = r
	s.Y.Rect = r
	if !r.Empty() {
		s.X.Points = make([][]int, r.Dy())
		s.Y.Points = make([][]int, r.Dx())
		for i := 0; i < len(s.X.Points); i++ {
			s.X.Points[i] = []int{r.Max.X, Range}
		}
		for i := 0; i < len(s.Y.Points); i++ {
			s.Y.Points[i] = []int{r.Max.Y, Range}
		}
	}
	return s
}
