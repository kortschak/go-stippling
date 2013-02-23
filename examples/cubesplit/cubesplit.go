/*
Like Split, but extends across the Z axis, which normally represents time. Expects a directory full
of frames of identical size. Splits density in half among longest axis. Repeats for sub-cells for g generations.
Incredibly memory hungry! Keep it low-res, low amount of frames if possible.
*/
package main

import (
	"code.google.com/p/go-stippling/density"
	"code.google.com/p/go-stippling/examples/util"
	"code.google.com/p/intmath/intgr"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

func main() {

	util.Init()

	files := util.ListFiles()

	if util.Mono {
		var sp *splitmap

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

		fmt.Printf("\nSplitting cells.\n")
		for i := 0; i < util.Generations; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tCells: %v\n", i, len(sp.cells))
		}

		fmt.Printf("\nConverting cells to cellstream.\n")
		cs := cellstreamFrom(sp)

		fmt.Printf("\nConverting cells back to %v frames.\n", sp.source.LenZ)

		imgout := image.NewGray16(sp.source.Rect)
		waitchan := make(chan int, util.MaxGoroutines)

		for z, i := 0, 0; z < sp.source.LenZ; z++ {
			for i := 0; i < util.MaxGoroutines; i++ {
				waitchan <- 1
			}
			for c := cs.stream[i]; c.Z == z; {
				_ = <-waitchan
				go func(c *streamcell) {
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
		var sp *splitmap

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

		fmt.Printf("\nSplitting cells.\n")

		for i := 0; i < util.GenerationsR; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tRed cells: %v\n", i, len(sp.cells))
		}

		fmt.Printf("\nConverting Red cells to cellstream.\n")
		rcs := cellstreamFrom(sp)

		fmt.Printf("\nFilling the cube with Green channels.\n")
		sp.source.LenZ = 0
		sp.cells = []*cell{&cell{
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

		fmt.Printf("\nSplitting cells.\n")
		for i := 0; i < util.GenerationsG; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tGreen cells: %v\n", i, len(sp.cells))
		}

		fmt.Printf("\nConverting Green cells to cellstream.\n")
		gcs := cellstreamFrom(sp)

		fmt.Println("\nFilling the cube with frames' Blue channels.\n")
		sp.source.LenZ = 0
		sp.cells = []*cell{&cell{
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

		fmt.Printf("\nSplitting cells.\n")
		for i := 0; i < util.GenerationsB; i++ {
			sp.Split()
			fmt.Printf("Generation: %v\tBlue cells: %v\n", i, len(sp.cells))
		}

		fmt.Printf("\nConverting Blue cells to cellstream.\n")
		bcs := cellstreamFrom(sp)
		sp.cells = []*cell{&cell{
			Source: sp.source,
			Rect:   sp.source.Rect,
			zmin:   0,
			zmax:   0,
			c:      0}}

		fmt.Printf("\nConverting cellstreams back to %v frames.\n", sp.source.LenZ)

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
				go func(c *streamcell) {
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
				go func(c *streamcell) {
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
				go func(c *streamcell) {
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

func listFiles(path string) []string {
	files := []string{}
	list := func(filepath string, f os.FileInfo, err error) error {
		files = append(files, filepath)
		return nil
	}
	err := filepath.Walk(path, list)
	if err != nil {
		log.Printf("filepath.Walk() returned %v\n", err)
	}
	return files
}

func fileToImage(fileName string) (img *image.Image, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer file.Close()

	image, _, err := image.Decode(file)
	if err != nil {
		log.Println(err, "Could not decode image:", fileName)
	}
	return &image, err
}

func sequence(outputName *string, frameNum int, ext *uint) string {

	splitName := *outputName + "-" + strconv.Itoa(frameNum)
	switch *ext {
	case 1:
		splitName = splitName + ".png"
	case 2:
		splitName = splitName + ".jpg"
	}
	return splitName
}

func imgToFile(i image.Image, filename string, ext *uint, jpgQuality *int) {

	output, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	switch *ext {
	case 1:
		png.Encode(output, i)
	case 2:
		jpeg.Encode(output, i, &jpeg.Options{*jpgQuality})
	}
}

type cell struct {
	Source     *density.CubeSum
	Rect       image.Rectangle
	zmin, zmax int
	c          uint16
}

func (c *cell) Mass() uint64 {
	return c.Source.Sum(c.Rect, c.zmin, c.zmax)
}

func (c *cell) CalcC() {
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

// Splits current cell - modifies itself to keep half of
// the current mass of the cell, returns other half as new cell
func (c *cell) Split(cellchan chan *cell) int {

	child := &cell{
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
		cellchan <- child
		return 1
	}
	if dx >= dy && dx > util.Xweight {
		c.Rect.Max.X = (cx + ncx + 1) / 2
		child.Rect.Min.X = (cx + ncx + 1) / 2
		cellchan <- child
		return 1
	}
	if dy > util.Yweight {
		c.Rect.Max.Y = (cy + ncy + 1) / 2
		child.Rect.Min.Y = (cy + ncy + 1) / 2
		cellchan <- child
		return 1
	}
	return 0
}

type splitmap struct {
	source *density.CubeSum
	cells  []*cell
}

func (sp *splitmap) Split() {
	cellchan := make(chan *cell, len(sp.cells)*2)
	waitchan := make(chan int, util.MaxGoroutines)

	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 0
	}
	var totalcells int
	for i, c := range sp.cells {
		totalcells += <-waitchan
		go func(i int, c *cell) {
			waitchan <- c.Split(cellchan)
		}(i, c)
	}
	for i := 0; i < util.MaxGoroutines; i++ {
		totalcells += <-waitchan
	}

	for i := 0; i < totalcells; i++ {
		sp.cells = append(sp.cells, <-cellchan)
	}

	return
}

func SPFrom(img *image.Image, capz int, dm density.Model) (sp *splitmap) {
	sp = new(splitmap)
	sp.source = density.CubeSumFrom(img, dm, capz)
	sp.cells = []*cell{&cell{
		Source: sp.source,
		Rect:   sp.source.Rect,
		zmin:   0,
		zmax:   1,
		c:      0}}
	return
}

func (sp *splitmap) AddFrame(i *image.Image, dm density.Model) {
	sp.source.AddFrame(i, dm)
	sp.cells[0].zmax = sp.source.LenZ
}

// Alternative ways to write to an image - cellstream is preferred
func (sp *splitmap) To(img *image.Gray16, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.cells {
		// If the cell overlaps with the frame...
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *cell) {
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

func (sp *splitmap) ToR(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *cell) {
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

func (sp *splitmap) ToG(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *cell) {
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

func (sp *splitmap) ToB(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *cell) {
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

func (sp *splitmap) ToA(img *image.RGBA, frame int) {
	waitchan := make(chan int, util.MaxGoroutines)
	for i := 0; i < util.MaxGoroutines; i++ {
		waitchan <- 1
	}
	for _, c := range sp.cells {
		if c.zmin <= frame && frame < c.zmax {
			_ = <-waitchan
			go func(c *cell) {
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
type streamcell struct {
	Z    int
	Rect image.Rectangle
	c    uint16
}

// A way to save a stream of cells. We only need to save zmin, because the bottom of the cube will
// be written over by the next cube anyway (similarly, when drawing these cubes only the part that
// changes needs redrawing).
type cellstream struct {
	stream []*streamcell
}

func (cs *cellstream) Append(c *cell) {
	c.CalcC()
	sc := &streamcell{
		Z:    c.zmin,
		Rect: c.Rect,
		c:    c.c,
	}
	cs.stream = append(cs.stream, sc)
	return
}

func cellstreamFrom(sp *splitmap) (cs *cellstream) {
	cs = new(cellstream)
	cs.stream = make([]*streamcell, 0, len(sp.cells))
	for _, c := range sp.cells {
		cs.Append(c)
	}
	sort.Sort(cs)
	return
}

func (cs *cellstream) Len() int {
	return len(cs.stream)
}

func (cs *cellstream) Less(i, j int) bool {
	return cs.stream[i].Z < cs.stream[j].Z
}

func (cs *cellstream) Swap(i, j int) {
	cs.stream[i], cs.stream[j] = cs.stream[j], cs.stream[i]
	return
}
