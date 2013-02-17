// Quarters density in half among centre of mass.
// Repeats for sub-cells for g generations.
package main

import (
	"code.google.com/p/go-stippling/density"
	"flag"
	//"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

const (
	maxGoRoutines = 8 //Low, to save memory overhead (gah, I need to learn how mutexes work...)
	Range         = 0xFFFF
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

		quarterName := *outputName + "-" + strconv.Itoa(fileNum) + "-"
		switch *outputExt {
		case 1:
			quarterName = quarterName + num + ".png"
		case 2:
			quarterName = quarterName + num + ".jpg"
		}
		return quarterName
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
			qrt := QRTFrom(img)
			imgout := image.NewGray16(qrt.source.X.Rect)
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					qrt.To(imgout)
					toFile(imgout, name(i))
				}
				qrt.Quarter()
			}
			qrt.To(imgout)
			toFile(imgout, name(*generations))
		} else {
			cqrt := CQRTFrom(img)
			imgout := image.NewRGBA(cqrt.R.source.X.Rect)
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					cqrt.To(imgout)
					toFile(imgout, name(i))
				}
				cqrt.Quarter()
			}
			cqrt.To(imgout)
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
	Source *density.Sum
	Mask   *density.SumMask
	c      uint16
}

func (c *cell) CalcC() {
	c.Mask.ApplyTo(c.Source)
	if c.Mask.X.MaskMass != 0 {
		c.c = uint16(0xFFFF * c.Mask.X.Mass / c.Mask.X.MaskMass)
	} else {
		c.c = 0
	}
}

// Quarters current cell - modifies itself to keep a quarter of
// the current mass of the cell, returns three new cells
func (c *cell) Quarter(cellchan chan *cell) int {
	r := c.Mask.X.Rect
	c.Mask.ApplyTo(c.Source)
	cx := int(math.Floor(c.Mask.WX() + 0.5))
	cy := int(math.Floor(c.Mask.WY() + 0.5))
	if cx < r.Min.X {
		cx = r.Min.X
	}
	if cy < r.Min.Y {
		cy = r.Min.Y
	}
	r1 := image.Rect(r.Min.X, r.Min.Y, cx, cy)
	r2 := image.Rect(cx, r.Min.Y, r.Max.X, cy)
	r3 := image.Rect(r.Min.X, cy, cx, r.Max.Y)
	r4 := image.Rect(cx, cy, r.Max.X, r.Max.Y)

	var totalcells int
	if !r1.Empty() {
		cellchan <- &cell{
			Source: c.Source,
			Mask:   density.NewSumMask(r1, Range),
			c:      0,
		}
		totalcells++
	}
	if !r2.Empty() {
		cellchan <- &cell{
			Source: c.Source,
			Mask:   density.NewSumMask(r2, Range),
			c:      0,
		}
		totalcells++
	}
	if !r3.Empty() {
		cellchan <- &cell{
			Source: c.Source,
			Mask:   density.NewSumMask(r3, Range),
			c:      0,
		}
		totalcells++
	}
	if !r4.Empty() {
		cellchan <- &cell{
			Source: c.Source,
			Mask:   density.NewSumMask(r4, Range),
			c:      0,
		}
		totalcells++
	}

	return totalcells
}

type quartmap struct {
	source *density.Sum
	cells  []*cell
}

func (qrt *quartmap) Quarter() {
	cellchan := make(chan *cell, len(qrt.cells)*4)
	waitchan := make(chan int, maxGoRoutines)

	for i := 0; i < maxGoRoutines; i++ {
		waitchan <- 0
	}
	var totalcells int
	for i, c := range qrt.cells {
		totalcells += <-waitchan
		go func(i int, c *cell) {

			waitchan <- c.Quarter(cellchan)
		}(i, c)
	}
	// Wait untill Quartering is done
	for i := 0; i < maxGoRoutines; i++ {
		totalcells += <-waitchan
	}
	newcells := make([]*cell, 0)
	for i := 0; i < totalcells; i++ {
		newcells = append(newcells, <-cellchan)
	}
	qrt.cells = newcells
	/*
		for i, v := range qrt.cells {
			v.CalcC()
			fmt.Printf("qrtQuarter %v\t Mass: %v\t C: %v\t W: %v-%v\t H: %v-%v\n", i, v.Mass(), v.c, v.r.Min.X, v.r.Max.X, v.r.Min.Y, v.r.Max.Y)
		}
		println()
	*/
	return
}

func (qrt *quartmap) To(img *image.Gray16) {
	waitchan := make(chan int, maxGoRoutines)
	for i := 0; i < maxGoRoutines; i++ {
		waitchan <- 1
	}
	for _, c := range qrt.cells {
		// since none of the pixels of the cells overlap, no worries about data races, right?
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			r := c.Mask.X.Rect
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
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

func QRTFrom(img image.Image) (qrt *quartmap) {
	qrt = new(quartmap)
	qrt.source = density.SumFrom(img, density.AvgDensity)
	qrt.cells = []*cell{&cell{
		Source: qrt.source,
		Mask:   density.NewSumMask(qrt.source.X.Rect, Range),
		c:      0}}
	return
}

type colorquartmap struct {
	R, G, B, A quartmap
}

func (cqrt *colorquartmap) Quarter() {
	cqrt.R.Quarter()
	cqrt.G.Quarter()
	cqrt.B.Quarter()
	cqrt.A.Quarter()
}

func CQRTFrom(img image.Image) (cqrt *colorquartmap) {
	cqrt = new(colorquartmap)
	cqrt.R.source = density.SumFrom(img, density.RedDensity)
	cqrt.G.source = density.SumFrom(img, density.GreenDensity)
	cqrt.B.source = density.SumFrom(img, density.BlueDensity)
	cqrt.A.source = density.SumFrom(img, density.AlphaDensity)

	cqrt.R.cells = []*cell{&cell{
		Source: cqrt.R.source,
		Mask:   density.NewSumMask(cqrt.R.source.X.Rect, Range),
		c:      0}}
	cqrt.G.cells = []*cell{&cell{
		Source: cqrt.G.source,
		Mask:   density.NewSumMask(cqrt.G.source.X.Rect, Range),
		c:      0}}
	cqrt.B.cells = []*cell{&cell{
		Source: cqrt.B.source,
		Mask:   density.NewSumMask(cqrt.B.source.X.Rect, Range),
		c:      0}}
	cqrt.A.cells = []*cell{&cell{
		Source: cqrt.A.source,
		Mask:   density.NewSumMask(cqrt.A.source.X.Rect, Range),
		c:      0}}
	return
}

func (cqrt *colorquartmap) To(img *image.RGBA) {
	waitchan := make(chan int, maxGoRoutines)
	for i := 0; i < maxGoRoutines; i++ {
		waitchan <- 1
	}
	for _, c := range cqrt.R.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			r := c.Mask.X.Rect
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4] = uint8(c.c >> 8)
				}
			}
			waitchan <- 1
		}(c)
	}
	for _, c := range cqrt.G.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			r := c.Mask.X.Rect
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4] = uint8(c.c >> 8)
				}
			}
			waitchan <- 1
		}(c)
	}
	for _, c := range cqrt.B.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			r := c.Mask.X.Rect
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4] = uint8(c.c >> 8)
				}
			}
			waitchan <- 1
		}(c)
	}
	for _, c := range cqrt.A.cells {
		_ = <-waitchan
		go func(c *cell) {
			c.CalcC()
			r := c.Mask.X.Rect
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					img.Pix[(y-img.Rect.Min.Y)*img.Stride+(x-img.Rect.Min.X)*4] = uint8(c.c >> 8)
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
