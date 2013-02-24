/*
Like Split, but extends across the Z axis, which normally represents time. Expects a directory full
of frames of identical size. Splits density in half among longest axis. Repeats for sub-Cells for g generations.
Incredibly memory hungry! Keep it low-res, low amount of frames if possible.
*/
package main

import (
	"code.google.com/p/go-stippling/density"
	"code.google.com/p/go-stippling/examples/util"
	"code.google.com/p/intmath/intgr"
	"fmt"
	"image"
	"log"
	"sort"
)

func main() {

	util.Init()

	files := util.ListFiles()

	if util.Mono {
		var sp *Map

		fmt.Printf("\nFilling the cube with frames.\n")
		for _, fileName := range files {
			img, err := util.FileToImage(fileName)
			if err == nil {
				if sp == nil {
					sp = SPFrom(img, len(files), density.AvgDensity)
				} else {
					fmt.Printf(".")
					sp.AddFrame(img, density.AvgDensity)
				}
			}
		}
		if sp == nil {
			log.Fatalf("Empty cube - ending program")
		}

		fmt.Printf("\nSplitting Cells.\n")
		for i := 0; i < util.Generations; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tCells: %v\n", i, len(sp.Cells))
		}

		fmt.Printf("\nConverting Cells to Cellstream.\n")
		cs := CellstreamFrom(sp)

		fmt.Printf("\nConverting Cells back to %v frames.\n", sp.source.LenZ)

		imgout := image.NewGray16(sp.source.Rect)
		waitchan := make(chan int, util.MaxGoroutines)

		for z, i := 0, 0; z < sp.source.LenZ; z++ {
			for i := 0; i < util.MaxGoroutines; i++ {
				waitchan <- 1
			}
			for c := cs.stream[i]; c.Z == z; {
				_ = <-waitchan
				go func(c *StreamCell) {
					for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
						for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
							j := imgout.PixOffset(x, y)
							imgout.Pix[j+0] = uint8(c.c >> 8)
							imgout.Pix[j+1] = uint8(c.c)
						}
					}
					waitchan <- 1
				}(c)
				if i < len(cs.stream)-1 {
					i++
					c = cs.stream[i]
				} else {
					break
				}
			}
			for i := 0; i < util.MaxGoroutines; i++ {
				_ = <-waitchan
			}

			util.ImgToFile(imgout, z)
			fmt.Printf(".")
		}
		fmt.Printf("done.")
	} else {
		var sp *Map

		fmt.Printf("\nFilling the cube with Red channels.\n")
		for _, fileName := range files {
			img, err := fileToImage(fileName)
			if err == nil {
				if sp == nil {
					sp = SPFrom(img, len(files), density.RedDensity)
				} else {
					fmt.Printf(".")
					sp.AddFrame(img, density.RedDensity)
				}
			}
		}

		fmt.Printf("\nSplitting Cells.\n")

		for i := 0; i < util.GenerationsR; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tRed Cells: %v\n", i, len(sp.Cells))
		}

		fmt.Printf("\nConverting Red Cells to Cellstream.\n")
		rcs := CellstreamFrom(sp)

		fmt.Printf("\nFilling the cube with Green channels.\n")
		sp.source.LenZ = 0
		sp.Cells = []*Cell{&Cell{
			Source: sp.source,
			Rect:   sp.source.Rect,
			zmin:   0,
			zmax:   0,
			c:      0}}

		for _, fileName := range files {
			img, err := fileToImage(fileName)
			if err == nil {
				fmt.Printf(".")
				sp.AddFrame(img, density.GreenDensity)
			}
		}

		fmt.Printf("\nSplitting Cells.\n")
		for i := 0; i < util.GenerationsG; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tGreen Cells: %v\n", i, len(sp.Cells))
		}

		fmt.Printf("\nConverting Green Cells to Cellstream.\n")
		gcs := CellstreamFrom(sp)

		fmt.Println("\nFilling the cube with frames' Blue channels.\n")
		sp.source.LenZ = 0
		sp.Cells = []*Cell{&Cell{
			Source: sp.source,
			Rect:   sp.source.Rect,
			zmin:   0,
			zmax:   0,
			c:      0}}
		for _, fileName := range files {
			img, err := fileToImage(fileName)
			if err == nil {
				fmt.Printf(".")
				sp.AddFrame(img, density.BlueDensity)
			}
		}

		fmt.Printf("\nSplitting Cells.\n")
		for i := 0; i < util.GenerationsB; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tBlue Cells: %v\n", i, len(sp.Cells))
		}

		fmt.Printf("\nConverting Blue Cells to Cellstream.\n")
		bcs := CellstreamFrom(sp)
		sp.Cells = []*Cell{&Cell{
			Source: sp.source,
			Rect:   sp.source.Rect,
			zmin:   0,
			zmax:   0,
			c:      0}}

		fmt.Printf("\nConverting Cellstreams back to %v frames.\n", sp.source.LenZ)

		imgout := image.NewRGBA(sp.source.Rect)

		//Make opaque
		for i := 3; i < len(imgout.Pix); i += 4 {
			imgout.Pix[i] = 0xFF
		}

		waitchan := make(chan int, util.MaxGoroutines)
		for z, r, g, b := 0, 0, 0, 0; z < sp.source.LenZ; z++ {
			for j := 0; j < util.MaxGoroutines; j++ {
				waitchan <- 1
			}
			for c := rcs.stream[r]; c.Z == z; {
				_ = <-waitchan
				go func(c *StreamCell) {
					for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
						for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
							imgout.Pix[(y-imgout.Rect.Min.Y)*imgout.Stride+(x-imgout.Rect.Min.X)*4] = uint8(c.c >> 8)
						}
					}
					waitchan <- 1
				}(c)
				if r < len(rcs.stream)-1 {
					r++
					c = rcs.stream[r]
				} else {
					break
				}
			}

			for c := gcs.stream[g]; c.Z == z; {
				_ = <-waitchan
				go func(c *StreamCell) {
					for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
						for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
							imgout.Pix[(y-imgout.Rect.Min.Y)*imgout.Stride+(x-imgout.Rect.Min.X)*4+1] = uint8(c.c >> 8)
						}
					}
					waitchan <- 1
				}(c)
				if g < len(gcs.stream)-1 {
					g++
					c = gcs.stream[g]
				} else {
					break
				}
			}

			for c := bcs.stream[b]; c.Z == z; {
				_ = <-waitchan
				go func(c *StreamCell) {
					for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
						for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
							imgout.Pix[(y-imgout.Rect.Min.Y)*imgout.Stride+(x-imgout.Rect.Min.X)*4+2] = uint8(c.c >> 8)
						}
					}
					waitchan <- 1
				}(c)
				if b < len(bcs.stream)-1 {
					b++
					c = bcs.stream[b]
				} else {
					break
				}
			}

			for j := 0; j < util.MaxGoroutines; j++ {
				_ = <-waitchan
			}

			util.ImgToFile(imgout, z)
			fmt.Printf(".")
		}
	}
	fmt.Printf("\ndone.")
}

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
// the current mass of the Cell, returns other half as new Cell
func (c *Cell) Split(Cellchan chan *Cell) int {

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
	if dz >= dx && dz >= dy && dz > util.Zweight {
		c.zmax = (cz + ncz + 1) / 2
		child.zmin = (cz + ncz + 1) / 2
		Cellchan <- child
		return 1
	}
	if dx >= dy && dx > util.Xweight {
		c.Rect.Max.X = (cx + ncx + 1) / 2
		child.Rect.Min.X = (cx + ncx + 1) / 2
		Cellchan <- child
		return 1
	}
	if dy > util.Yweight {
		c.Rect.Max.Y = (cy + ncy + 1) / 2
		child.Rect.Min.Y = (cy + ncy + 1) / 2
		Cellchan <- child
		return 1
	}
	return 0
}

type Map struct {
	source *density.CubeSum
	Cells  []*Cell
}

func (sp *Map) Split() {
	Cellchan := make(chan *Cell, len(sp.Cells)*2)
	waitchan := make(chan int, util.MaxGoroutines)

	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 0
	}
	var totalCells int
	for i, c := range sp.Cells {
		totalCells += <-waitchan
		go func(i int, c *Cell) {
			waitchan <- c.Split(Cellchan)
		}(i, c)
	}
	for i := 0; i < util.MaxGoroutines; i++ {
		totalCells += <-waitchan
	}

	for i := 0; i < totalCells; i++ {
		sp.Cells = append(sp.Cells, <-Cellchan)
	}

	return
}

func SPFrom(img *image.Image, capz int, dm density.Model) (sp *Map) {
	sp = new(Map)
	sp.source = density.CubeSumFrom(img, dm, capz)
	sp.Cells = []*Cell{&Cell{
		Source: sp.source,
		Rect:   sp.source.Rect,
		zmin:   0,
		zmax:   1,
		c:      0}}
	return
}

func (sp *Map) AddFrame(i *image.Image, dm density.Model) {
	sp.source.AddFrame(i, dm)
	sp.Cells[0].zmax = sp.source.LenZ
}

// Alternative ways to write to an image - Cellstream is preferred
func (sp *Map) To(img *image.Gray16, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.Cells {
		// If the Cell overlaps with the frame...
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *Cell) {
				for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
					for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
						i := img.PixOffset(x, y)
						img.Pix[i+0] = uint8(c.c >> 8)
						img.Pix[i+1] = uint8(c.c)
					}
				}
				waitchan <- 1
			}(c)
		}
	}
	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	return
}

func (sp *Map) ToR(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.Cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *Cell) {
				for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
					for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
						img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4] = uint8(c.c >> 8)
					}
				}
				waitchan <- 1
			}(c)
		}
	}

	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	return
}

func (sp *Map) ToG(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.Cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *Cell) {
				for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
					for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
						img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4+1] = uint8(c.c >> 8)
					}
				}
				waitchan <- 1
			}(c)
		}
	}

	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	return
}

func (sp *Map) ToB(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.Cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *Cell) {
				for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
					for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
						img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4+2] = uint8(c.c >> 8)
					}
				}
				waitchan <- 1
			}(c)
		}
	}

	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	return
}

func (sp *Map) ToA(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.Cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *Cell) {
				for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
					for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
						img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4+3] = uint8(c.c >> 8)
					}
				}
				waitchan <- 1
			}(c)
		}
	}

	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	return
}

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
