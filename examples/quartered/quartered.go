// Splits density in half among longest axis.
// Repeats for sub-cells for g generations.
// Does not do sub-pixel precision.
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
	"runtime"
	"strconv"
)

const (
	maxGoRoutines = 256
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
	var inputfiles = flag.Args()

	if *numCores <= 0 || *numCores > runtime.NumCPU() {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*numCores)
	}

	var fileNum int
	var fileName string

	// Use a function variable for processing the files, so that defer
	// gets called for closing the files without having to pass all of
	// the variables. This admittedly feels dirty, but it works.
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

	processFiles := func() {
		file, err := os.Open(fileName)
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			log.Println(err, "\tfilename:", fileName, "\tinput nr:", fileNum)
			return
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
			qm := QMFrom(img)
			imgout := image.NewGray16(qm.ds.Rect)
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					qm.To(imgout)
					toFile(imgout, name(i))
				}
				qm.Split()
			}
			qm.To(imgout)
			toFile(imgout, name(*generations))
		} else {
			cqm := CQMFrom(img)
			imgout := image.NewRGBA(cqm.R.ds.Rect)
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					cqm.To(imgout)
					toFile(imgout, name(i))
				}
				cqm.Split()
			}
			cqm.To(imgout)
			toFile(imgout, name(*generations))
		}
	}

	for fileNum, fileName = range inputfiles {
		processFiles()

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

type quartermap struct {
	ds    *density.DSum
	cells []*cell
}

func (qm *quartermap) Split() {
	qm.cells = append(qm.cells, qm.cells...)
	oldcells := qm.cells[:len(qm.cells)/2]
	newcells := qm.cells[len(qm.cells)/2:]
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
		for i, v := range qm.cells {
			v.CalcC()
			fmt.Printf("QMSplit %v\t Mass: %v\t C: %v\t W: %v-%v\t H: %v-%v\n", i, v.Mass(), v.c, v.r.Min.X, v.r.Max.X, v.r.Min.Y, v.r.Max.Y)
		}
		println()
	*/
	return
}

func (qm *quartermap) To(img *image.Gray16) {
	waitchan := make(chan int, maxGoRoutines)
	for i := 0; i < maxGoRoutines; i++ {
		waitchan <- 1
	}
	for _, c := range qm.cells {
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

func QMFrom(img image.Image) (qm *quartermap) {
	qm = new(quartermap)
	qm.ds = density.DSumFrom(img, density.AvgDensity)
	qm.cells = []*cell{&cell{
		Source: qm.ds,
		r:      qm.ds.Rect,
		c:      0}}
	return
}

type colorquartermap struct {
	R, G, B, A quartermap
}

func (cqm *colorquartermap) Split() {
	cqm.R.Split()
	cqm.G.Split()
	cqm.B.Split()
	cqm.A.Split()
}

func CQMFrom(img image.Image) (cqm *colorquartermap) {
	cqm = new(colorquartermap)
	cqm.R.ds = density.DSumFrom(img, density.RedDensity)
	cqm.G.ds = density.DSumFrom(img, density.GreenDensity)
	cqm.B.ds = density.DSumFrom(img, density.BlueDensity)
	cqm.A.ds = density.DSumFrom(img, density.AlphaDensity)

	cqm.R.cells = []*cell{&cell{
		Source: cqm.R.ds,
		r:      cqm.R.ds.Rect,
		c:      0}}
	cqm.G.cells = []*cell{&cell{
		Source: cqm.G.ds,
		r:      cqm.G.ds.Rect,
		c:      0}}
	cqm.B.cells = []*cell{&cell{
		Source: cqm.B.ds,
		r:      cqm.B.ds.Rect,
		c:      0}}
	cqm.A.cells = []*cell{&cell{
		Source: cqm.A.ds,
		r:      cqm.A.ds.Rect,
		c:      0}}
	return
}

func (cqm *colorquartermap) To(img *image.RGBA) {
	waitchan := make(chan int, maxGoRoutines)
	for i := 0; i < maxGoRoutines; i++ {
		waitchan <- 1
	}
	for _, c := range cqm.R.cells {
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
	for _, c := range cqm.G.cells {
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
	for _, c := range cqm.B.cells {
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
	for _, c := range cqm.A.cells {
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
