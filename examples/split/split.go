// Splits density in half among longest axis.
// Repeats for sub-cells for g generations.
// Does not do sub-pixel precision, so it
// will get "stuck" (try splitting into
// more cells than the original number
// of pixels to see this in practice).
package main

import (
	"code.google.com/p/go-stippling/density"
	"flag"
	//"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

const (
	maxGoRoutines = 256 //Arbitrarily chosen
)

func main() {

	var outputName = flag.String("o", "output", "\t\tName of the (o)utput (no extension)")
	var outputExt = flag.Uint("e", 1, "\t\tOutput (e)xtension type:\n\t\t\t 1 \t png (default)\n\t\t\t 2 \t jpg")
	var jpgQuality = flag.Int("q", 90, "\t\tJPG output (q)uality")
	var generations = flag.Uint("g", 3, "\t\tNumber of (g)enerations")
	var saveAll = flag.Bool("s", true, "\t\t(s)ave all generations (default) - only save last generation if false")
	var numCores = flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	var mono = flag.Bool("mono", true, "\t\tMonochrome or colour output")
	flag.Parse()

	if *numCores <= 0 || *numCores > runtime.NumCPU() {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*numCores)
	}

	// Use a function variable for processing the files, so that defer
	// gets called for closing the files without having to pass all of
	// the variables. This admittedly feels dirty, but it works.
	var fileNum int

	name := func(g uint) string {
		num := ""
		// I highly doubt anyone would try to go beyond 99 generations,
		// as that would generate over 2^99 cells.
		if g%10 == g {
			num = num + "0"
		}
		num = num + strconv.Itoa(int(g))

		splitName := *outputName + "-" + strconv.Itoa(fileNum) + "-"
		switch *outputExt {
		case 1:
			splitName = splitName + num + ".png"
		case 2:
			splitName = splitName + num + ".jpg"
		}
		return splitName
	}

	processFiles := func(fileName string) error {
		file, err := os.Open(fileName)
		if err != nil {
			log.Println(err)
			return err
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			log.Println(err, "Could not decode image:", fileName)
			return nil
		}
		toFile := func(i image.Image, outputname string) {

			output, err := os.Create(outputname)
			if err != nil {
				log.Fatal(err)
			}
			defer output.Close()

			switch *outputExt {
			case 1:
				png.Encode(output, i)
			case 2:
				jpeg.Encode(output, i, &jpeg.Options{*jpgQuality})
			}
		}

		if *mono {
			sp := SPFrom(img)
			imgout := image.NewGray16(sp.ds.Rect)
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					sp.To(imgout)
					toFile(imgout, name(i))
				}
				sp.Split()
			}
			sp.To(imgout)
			toFile(imgout, name(*generations))
		} else {
			csp := CSPFrom(img)
			imgout := image.NewRGBA(csp.R.ds.Rect)
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					csp.To(imgout)
					toFile(imgout, name(i))
				}
				csp.Split()
			}
			csp.To(imgout)
			toFile(imgout, name(*generations))
		}
		fileNum++
		return nil
	}

	files := []string{}
	listFiles := func(path string, f os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	}

	root := flag.Arg(0)
	err := filepath.Walk(root, listFiles)
	if err != nil {
		log.Printf("filepath.Walk() returned %v\n", err)
	}

	for _, file := range files {
		processFiles(file)
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

	if c.r.Dx() > c.r.Dy() {
		xmin := c.r.Min.X
		xmax := c.r.Max.X
		child.r.Max.X = (xmin + xmax) / 2
		for { //i := 0; ; i++ {
			if xmax-xmin > 1 {
				if child.Mass() < c.Mass()/2 {
					xmin = child.r.Max.X
					child.r.Max.X = (child.r.Max.X + xmax) / 2
				} else if child.Mass() > c.Mass()/2 {
					xmax = child.r.Max.X
					child.r.Max.X = (child.r.Max.X + xmin) / 2
				} else {
					c.r.Min.X = child.r.Max.X
					return
				}
				//fmt.Printf("i: %v \t x: %v\n", i, child.r.Max.X)
			} else {
				c.r.Min.X = child.r.Max.X
				return
			}
		}
	} else {
		ymin := c.r.Min.Y
		ymax := c.r.Max.Y
		child.r.Max.Y = (ymin + ymax) / 2
		for { //i := 0; ; i++ {
			if ymax-ymin > 1 {
				if child.Mass() < c.Mass()/2 {
					ymin = child.r.Max.Y
					child.r.Max.Y = (child.r.Max.Y + ymax) / 2
				} else if child.Mass() > c.Mass()/2 {
					ymax = child.r.Max.Y
					child.r.Max.Y = (child.r.Max.Y + ymin) / 2
				} else {
					c.r.Min.Y = child.r.Max.Y
					return
				}
				//fmt.Printf("i: %v \t y: %v\n", i, child.r.Max.Y)
			} else {
				c.r.Min.Y = child.r.Max.Y
				return
			}
		}

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
	waitchan := make(chan int, maxGoRoutines)

	for i := 0; i < maxGoRoutines; i++ {
		waitchan <- 1
	}
	for i, c := range oldcells {
		_ = <-waitchan
		go func(i int, c *cell) {
			newcells[i] = c.Split()
			waitchan <- 1
		}(i, c)
	}
	for i := 0; i < maxGoRoutines; i++ {
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
	waitchan := make(chan int, maxGoRoutines)
	for i := 0; i < maxGoRoutines; i++ {
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
	for i := 0; i < maxGoRoutines; i++ {
		_ = <-waitchan
	}
	return
}

func SPFrom(img image.Image) (sp *splitmap) {
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

func CSPFrom(img image.Image) (csp *colorsplitmap) {
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
	waitchan := make(chan int, maxGoRoutines)
	for i := 0; i < maxGoRoutines; i++ {
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
	for i := 0; i < maxGoRoutines; i++ {
		_ = <-waitchan
	}
	return
}
