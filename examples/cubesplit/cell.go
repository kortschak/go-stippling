package main

import (
	"github.com/kortschak/go-stippling/density"
	"github.com/kortschak/go-stippling/examples/util"
	"github.com/thomaso-mirodin/intmath/intgr"
	"image"
)

type Cell struct {
	Source     *density.CubeSum
	Rect       image.Rectangle
	zmin, zmax int
	c          uint16
}

func (c *Cell) Mass() uint64 {
	return c.Source.VolumeSum(c.Rect, c.zmin, c.zmax)
}

func (c *Cell) CalcC() {
	dx := uint64(c.Rect.Dx())
	dy := uint64(c.Rect.Dy())
	dz := uint64(c.zmax - c.zmin)
	if volume := dx * dy * dz; volume != 0 {
		c.c = uint16(c.Mass() / volume)
		//fmt.Printf("\nc.Mass() %v\t volume %v\n", c.Mass(), volume)
	} else {
		c.c = 0
		//fmt.Printf("\nVolume empty! %v\n", c.c)
	}
}

// Splits current Cell - modifies itself to keep half of
// the current mass of the Cell, turns other half into new Cell
// If split, sends new cell into splitchan. If not, send input
// to staticchan
// Returns number of new cells.
func (c *Cell) Split(splitchan, staticchan chan *Cell) int {

	child := &Cell{
		Source: c.Source,
		Rect:   c.Rect,
		zmax:   c.zmax,
		zmin:   c.zmin,
		c:      0,
	}

	cx := c.Source.FindCx(c.Rect, c.zmin, c.zmax)
	cy := c.Source.FindCy(c.Rect, c.zmin, c.zmax)
	cz := c.Source.FindCz(c.Rect, c.zmin, c.zmax)
	ncx := c.Source.FindNegCx(c.Rect, c.zmin, c.zmax)
	ncy := c.Source.FindNegCy(c.Rect, c.zmin, c.zmax)
	ncz := c.Source.FindNegCz(c.Rect, c.zmin, c.zmax)
	dx := util.Xweight * intgr.Abs(cx-ncx)
	dy := util.Yweight * intgr.Abs(cy-ncy)
	dz := util.Zweight * intgr.Abs(cz-ncz)
	if dz >= dx && dz >= dy && dz >= util.Zweight { //&& dz < util.Zweight*(c.zmax-c.zmin) {
		c.zmax = (cz + ncz + 1) / 2
		child.zmin = (cz + ncz + 1) / 2
		splitchan <- child
		splitchan <- c
		return 1
	}
	if dx >= dy && dx >= util.Xweight { //&& dx < util.Xweight*c.Rect.Dx() {
		c.Rect.Max.X = (cx + ncx + 1) / 2
		child.Rect.Min.X = (cx + ncx + 1) / 2
		splitchan <- child
		splitchan <- c
		return 1
	}
	if dy >= util.Yweight { //&& dy < util.Yweight&c.Rect.Dy() {
		c.Rect.Max.Y = (cy + ncy + 1) / 2
		child.Rect.Min.Y = (cy + ncy + 1) / 2
		splitchan <- child
		splitchan <- c
		return 1
	}
	staticchan <- c
	return 0
}
