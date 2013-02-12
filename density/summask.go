package density

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

func (sm *SumMask) Wx() float64 {
	return sm.Y.Wx
}

func (sm *SumMask) Wy() float64 {
	return sm.X.Wy
}
