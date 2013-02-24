package main

import (
	"image"
	"sort"
)

// A rectangle in in a Cellstream
type StreamCell struct {
	Z    int
	Rect image.Rectangle
	c    uint16
}

// A way to save a stream of Cells. We only need to save zmin, because the bottom of the cube will
// be written over by the next cube anyway (similarly, when drawing these cubes only the part that
// changes needs redrawing).
type Cellstream struct {
	stream []*StreamCell
}

func (cs *Cellstream) Append(c *Cell) {
	c.CalcC()
	sc := &StreamCell{
		Z:    c.zmin,
		Rect: c.Rect,
		c:    c.c,
	}
	cs.stream = append(cs.stream, sc)
	return
}

func CellstreamFrom(sp *Map) (cs *Cellstream) {
	cs = new(Cellstream)
	cs.stream = make([]*StreamCell, 0, len(sp.Cells))
	for _, c := range sp.Cells {
		cs.Append(c)
	}
	for _, c := range sp.StaticCells {
		cs.Append(c)
	}
	sort.Sort(cs)
	return
}

func (cs *Cellstream) Len() int {
	return len(cs.stream)
}

func (cs *Cellstream) Less(i, j int) bool {
	return cs.stream[i].Z < cs.stream[j].Z
}

func (cs *Cellstream) Swap(i, j int) {
	cs.stream[i], cs.stream[j] = cs.stream[j], cs.stream[i]
	return
}
