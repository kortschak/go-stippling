// Splits density in half among longest axis.
// Repeats for sub-cells for g generations.
// Does not do sub-pixel precision, so it
// will get "stuck" (try splitting into
// more cells than the original number
// of pixels to see this in practice).
package main

import (
	"code.google.com/p/go-stippling/density"
	"code.google.com/p/go-stippling/examples/util"
	"code.google.com/p/intmath/intgr"
	"fmt"
	"image"
)

func main() {
	util.Init()
	files := util.ListFiles()

	if util.Mono {
		for filenum, fileName := range files {
			if util.Verbose {
				fmt.Printf("\nLoading file %s\n", fileName)
			}
			if img, err := util.FileToImage(fileName); err == nil {
				sp := SPFrom(img)
				imgout := image.NewGray16(sp.ds.Rect)

				if util.Verbose {
					fmt.Printf("\nSplitting Cells.\n")
				}
				for i := 0; i < util.Generations; i++ {
					if util.SaveAll {
						sp.To(imgout)
						util.ImgToFile(imgout, i, filenum)
					}
					sp.Split()
					if util.Verbose {
						fmt.Printf("Generation: %v\tCells: %v\n", i, len(sp.cells))
					}
				}
				sp.To(imgout)
				util.ImgToFile(imgout, util.Generations, filenum)
			}
		}
	} else {
		for filenum, fileName := range files {
			if util.Verbose {
				fmt.Printf("\nLoading file %s\n", fileName)
			}
			if img, err := util.FileToImage(fileName); err == nil {
				csp := CSPFrom(img)
				imgout := image.NewRGBA(csp.R.ds.Rect)

				if util.Verbose {
					fmt.Printf("\nSplitting Cells.\n")
				}
				for i := 0; i < util.Generations; i++ {
					if util.SaveAll {
						csp.To(imgout)
						util.ImgToFile(imgout,
							intgr.Min(i, util.GenerationR), intgr.Min(i, util.GenerationG),
							intgr.Min(i, util.GenerationB), intgr.Min(i, util.GenerationA),
							filenum)
					}
					if i < util.GenerationsR {
						csp.R.Split()
					}
					if i < util.GenerationsG {
						csp.G.Split()
					}
					if i < util.GenerationsB {
						csp.B.Split()
					}
					if i < util.GenerationsA {
						csp.A.Split()
					}
					if util.Verbose {
						fmt.Printf("Generation: %v\t Red Cells: %v\t Green Cells: %v\t Blue Cells: %v\t Alpha Cells: %v\n", i, len(csp.R.cells), len(csp.G.cells), len(csp.B.cells), len(csp.A.cells))
					}
				}
				csp.To(imgout)
				util.ImgToFile(imgout,
					intgr.Min(util.Generations, util.GenerationR), intgr.Min(util.Generations, util.GenerationG),
					intgr.Min(util.Generations, util.GenerationB), intgr.Min(util.Generations, util.GenerationA),
					filenum)
			}
		}
	}
}

type cell struct {
	Source *density.DSum
	r      image.Rectangle
	c      uint16
}

func (c *cell) Mass() uint64 {
	return c.Source.AreaSum(c.r)
}

func (c *cell) CalcC() {
	if c.r.Dx()*c.r.Dy() != 0 {
		c.c = uint16(c.Mass() / uint64(c.r.Dx()*c.r.Dy()))
	} else {
		c.c = 0
	}
}

// Splits current cell - modifies itself to keep half of
// the current mass of the cell, returns other half as new cell
func (c *cell) Split() (child *cell) {

	child = &cell{
		Source: c.Source,
		r:      c.r,
		c:      0,
	}

	if util.Xweight*c.r.Dx() > util.Yweight*c.r.Dy() {
		x := c.Source.FindCx(c.r)
		c.r.Max.X = x
		child.r.Min.X = x
	} else {
		y := c.Source.FindCy(c.r)
		c.r.Max.Y = y
		child.r.Min.Y = y
	}
	return
}

type splitmap struct {
	ds    *density.DSum
	cells []*cell
}

func (sp *splitmap) Split() {
	sp.cells = append(sp.cells, sp.cells...)
	oldcells := sp.cells[:len(sp.cells)/2]
	newcells := sp.cells[len(sp.cells)/2:]
	waitchan := make(chan int, util.MaxGoroutines)

	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for i, c := range oldcells {
		_ = <-waitchan
		go func(i int, c *cell) {
			newcells[i] = c.Split()
			waitchan <- 1
		}(i, c)
	}
	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	/*
		for i, v := range sp.cells {
			v.CalcC()
			fmt.Printf("spSplit %v\t Mass: %v\t C: %v\t W: %v-%v\t H: %v-%v\n", i, v.Mass(), v.c, v.r.Min.X, v.r.Max.X, v.r.Min.Y, v.r.Max.Y)
		}
		println()
	*/
	return
}

func (sp *splitmap) To(img *image.Gray16) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.cells {
		// since none of the pixels of the cells overlap, no worries about data races, right?
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			for y := c.r.Min.Y; y < c.r.Max.Y; y++ {
				for x := c.r.Min.X; x < c.r.Max.X; x++ {
					i := img.PixOffset(x, y)
					img.Pix[i+0] = uint8(c.c >> 8)
					img.Pix[i+1] = uint8(c.c)
				}
			}
			waitchan <- 1
		}(c)

	}
	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	return
}

func SPFrom(img *image.Image) (sp *splitmap) {
	sp = new(splitmap)
	sp.ds = density.DSumFrom(img, density.AvgDensity)
	sp.cells = []*cell{&cell{
		Source: sp.ds,
		r:      sp.ds.Rect,
		c:      0}}
	return
}

type colorsplitmap struct {
	R, G, B, A splitmap
}

func (csp *colorsplitmap) Split() {
	csp.R.Split()
	csp.G.Split()
	csp.B.Split()
	csp.A.Split()
}

func CSPFrom(img *image.Image) (csp *colorsplitmap) {
	csp = new(colorsplitmap)
	csp.R.ds = density.DSumFrom(img, density.RedDensity)
	csp.G.ds = density.DSumFrom(img, density.GreenDensity)
	csp.B.ds = density.DSumFrom(img, density.BlueDensity)
	csp.A.ds = density.DSumFrom(img, density.AlphaDensity)

	csp.R.cells = []*cell{&cell{
		Source: csp.R.ds,
		r:      csp.R.ds.Rect,
		c:      0}}
	csp.G.cells = []*cell{&cell{
		Source: csp.G.ds,
		r:      csp.G.ds.Rect,
		c:      0}}
	csp.B.cells = []*cell{&cell{
		Source: csp.B.ds,
		r:      csp.B.ds.Rect,
		c:      0}}
	csp.A.cells = []*cell{&cell{
		Source: csp.A.ds,
		r:      csp.A.ds.Rect,
		c:      0}}
	return
}

func (csp *colorsplitmap) To(img *image.RGBA) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range csp.R.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			for y := c.r.Min.Y; y < c.r.Max.Y; y++ {
				for x := c.r.Min.X; x < c.r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4] = uint8(c.c >> 8)
				}
			}
			waitchan <- 1
		}(c)
	}
	for _, c := range csp.G.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			for y := c.r.Min.Y; y < c.r.Max.Y; y++ {
				for x := c.r.Min.X; x < c.r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4+1] = uint8(c.c >> 8)
				}
			}
			waitchan <- 1
		}(c)
	}
	for _, c := range csp.B.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			for y := c.r.Min.Y; y < c.r.Max.Y; y++ {
				for x := c.r.Min.X; x < c.r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4+2] = uint8(c.c >> 8)
				}
			}
			waitchan <- 1
		}(c)
	}
	for _, c := range csp.A.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			for y := c.r.Min.Y; y < c.r.Max.Y; y++ {
				for x := c.r.Min.X; x < c.r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4+3] = uint8(c.c >> 8)
				}
			}
			waitchan <- 1
		}(c)
	}
	for i := 0; i < util.MaxGoroutines; i++ {
		_ = <-waitchan
	}
	return
}
