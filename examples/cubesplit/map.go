package main

import (
	"github.com/kortschak/go-stippling/density"
	"github.com/kortschak/go-stippling/examples/util"
	"image"
)

type Map struct {
	source *density.CubeSum
	// Seperate cells that no longer cubelit from those that do
	// to cubeeed up passes.
	Cells, StaticCells []*Cell
}

func (cube *Map) Split() {
	cubelitchan := make(chan *Cell, len(cube.Cells)*2)
	staticchan := make(chan *Cell, len(cube.Cells))
	waitchan := make(chan int, util.MaxGoroutines)

	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 0
	}
	var newCells int
	for i, c := range cube.Cells {
		newCells += <-waitchan
		go func(i int, c *Cell) {
			waitchan <- c.Split(cubelitchan, staticchan)
		}(i, c)
	}
	for i := 0; i < util.MaxGoroutines; i++ {
		newCells += <-waitchan
	}

	for i := 0; i < len(cube.Cells)-newCells; i++ {
		cube.StaticCells = append(cube.StaticCells, <-staticchan)
	}
	cube.Cells = cube.Cells[:0]
	for i := 0; i < 2*newCells; i++ {
		cube.Cells = append(cube.Cells, <-cubelitchan)
	}

	return
}

func NewCube(Rect image.Rectangle, capz int) *Map {
	cube := new(Map)
	cube.source = density.NewCubeSum(Rect, capz)
	cube.Cells = []*Cell{&Cell{
		Source: cube.source,
		Rect:   cube.source.Rect,
		zmin:   0,
		zmax:   0,
		c:      0}}
	return cube
}

func From(img *image.Image, dm density.Model, capz int) (cube *Map) {
	cube = new(Map)
	cube.source = density.CubeSumFrom(img, dm, capz)
	cube.Cells = []*Cell{&Cell{
		Source: cube.source,
		Rect:   cube.source.Rect,
		zmin:   0,
		zmax:   1,
		c:      0}}
	return
}

func (cube *Map) AddFrame(i *image.Image, dm density.Model) {
	cube.source.AddFrame(i, dm)
	cube.Cells[0].zmax = cube.source.LenZ
}

// Alternative way to write to a single frame to a Gray16 image - when converting the whole cube CellStream is preferred
func (cube *Map) To(img *image.Gray16, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range cube.Cells {
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
	for _, c := range cube.StaticCells {
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

func (cube *Map) ToR(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range cube.Cells {
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
	for _, c := range cube.StaticCells {
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

func (cube *Map) ToG(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range cube.Cells {
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
	for _, c := range cube.StaticCells {
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

func (cube *Map) ToB(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range cube.Cells {
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
	for _, c := range cube.StaticCells {
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

func (cube *Map) ToA(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range cube.Cells {
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
	for _, c := range cube.StaticCells {
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
