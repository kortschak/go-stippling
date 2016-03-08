/*
Like Split, but extends across the Z axis, which normally represents time. Expects a directory full
of frames of identical size. Splits density in half among longest axis. Repeats for sub-Cells for g generations.
Incredibly memory hungry! Keep it low-res, low amount of frames if possible.
*/
package main

import (
	"fmt"
	"github.com/kortschak/go-stippling/density"
	"github.com/kortschak/go-stippling/examples/util"
	"github.com/thomaso-mirodin/intmath/intgr"
	"image"
	"log"
)

func main() {

	util.Init()

	argsList := util.ListFiles()

	var frameNum int
	if util.Mono {
		for _, files := range argsList {
			frameNum = monoCube(files, frameNum)
		}
	} else {
		for _, files := range argsList {
			frameNum = colorCube(files, frameNum)
		}
	}
	fmt.Printf("\ndone.\n")
}

func monoCube(files []string, frameNum int) int {
	cube, cs := calcCube(files, nil, util.Generations, density.AvgDensity)
	if util.Verbose {
		fmt.Printf("\nConverting Cells back to %v frames.\n", cube.source.LenZ)
	}
	imgout := image.NewGray16(cube.source.Rect)
	waitchan := make(chan int, util.MaxGoroutines)

	for z, i := 0, 0; z < cube.source.LenZ; z++ {
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

		util.ImgToFile(imgout, util.Generations, z+frameNum)
		if util.Verbose {
			fmt.Printf(".")
		}
	}
	return frameNum + cube.source.LenZ
}

func colorCube(files []string, frameNum int) int {
	var cube *Map
	var rcs, gcs, bcs *Cellstream

	if util.Verbose {
		fmt.Printf("\n== RED CHANNEL ==\n")
	}
	cube, rcs = calcCube(files, cube, util.GenerationsR, density.RedDensity)
	if util.Verbose {
		fmt.Printf("\n== GREEN CHANNEL ==\n")
	}
	cube, gcs = calcCube(files, cube, util.GenerationsG, density.GreenDensity)
	if util.Verbose {
		fmt.Printf("\n== BLUE CHANNEL ==\n")
	}
	cube, bcs = calcCube(files, cube, util.GenerationsB, density.BlueDensity)

	if util.Verbose {
		fmt.Printf("\nConverting Cellstreams back to %v frames.\n", cube.source.LenZ)
	}

	imgout := image.NewRGBA(cube.source.Rect)
	waitchan := make(chan int, util.MaxGoroutines)
	//Make opaque
	for i := 3; i < len(imgout.Pix); i += 4 {
		imgout.Pix[i] = 0xFF
	}

	for z, r, g, b := 0, 0, 0, 0; z < cube.source.LenZ; z++ {
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

		util.ImgToFile(imgout,
			intgr.Min(util.Generations, util.GenerationsR), intgr.Min(util.Generations, util.GenerationsG),
			intgr.Min(util.Generations, util.GenerationsB), intgr.Min(util.Generations, util.GenerationsA),
			z+frameNum)
		if util.Verbose {
			fmt.Printf(".")
		}
	}
	return frameNum + cube.source.LenZ
}

func calcCube(files []string, cube *Map, gen int, dmodel density.Model) (cubeout *Map, cs *Cellstream) {

	if util.Verbose {
		fmt.Printf("\nFilling the cube with frames.\n")
	}
	if cube == nil {
		for i, file := range files {
			img, err := util.FileToImage(file)
			// keep going until first image is found
			if err == nil {
				cube = NewCube((*img).Bounds(), len(files)-i)
				break
			}
		}
		if cube == nil {
			log.Fatalf("Empty cube - ending program")
		}
	} else {
		// since we're passing over the same volume multiple times (once per channel)
		// we can reuse the allocated memory for the first cube.
		cube.source.LenZ = 0

		// Allow the old cells to be garbage collected
		for i := 0; i < len(cube.Cells); i++ {
			cube.Cells[i] = nil
		}
		cube.Cells = cube.Cells[:1]
		cube.Cells[0] = &Cell{
			Source: cube.source,
			Rect:   cube.source.Rect,
			zmin:   0,
			zmax:   0,
			c:      0}
		cube.StaticCells = nil
	}
	for _, fileName := range files {
		img, err := util.FileToImage(fileName)
		if err == nil {
			if util.Verbose {
				fmt.Printf(".")
			}
			cube.AddFrame(img, dmodel)
		}

	}

	if util.Verbose {
		fmt.Printf("\nSplitting Cells.\n")
	}
	for i := 0; i < gen; i++ {
		cube.Split()
		if util.Verbose {
			fmt.Printf("Generation: %v\tSplitting Cells: %v\tStable Cells: %v\n", i, len(cube.Cells), len(cube.StaticCells))
		}
	}

	if util.Verbose {
		fmt.Printf("\nConverting Cells to Cellstream.\n")
	}
	return cube, CellstreamFrom(cube)
}
