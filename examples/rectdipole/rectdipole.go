// Implements rectangular version of dipole.
// Does not do sub-pixel precision, so it
// will get "stuck" (try splitting into
// more cells than the original number
// of pixels to see this in practice).
package main

import (
	"code.google.com/p/go-stippling/density"
	"code.google.com/p/intmath/intgr"
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

		splitName := *outputName + "-" + num + "-" + strconv.Itoa(fileNum)
		switch *outputExt {
		case 1:
			splitName = splitName + ".png"
		case 2:
			splitName = splitName + ".jpg"
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
			imgout := image.NewGray16(sp.north.Rect)
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
			imgout := image.NewRGBA(csp.R.north.Rect)
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
	North, South *density.DSum
	Rect         image.Rectangle
	c            uint16
}

func (c *cell) CalcC() {
	if c.Rect.Dx()*c.Rect.Dy() != 0 {
		c.c = uint16(c.North.AreaSum(c.Rect) / uint64(c.Rect.Dx()*c.Rect.Dy()))
	} else {
		c.c = 0
	}
}

// Given a DSum and a Rectangle, finds x closest to line
// dividing mass in half.
// Does not do bounds checking - assumes r is inside ds.Rect
func findcx(ds *density.DSum, r image.Rectangle) int {
	xmin := r.Min.X
	xmax := r.Max.X
	x := (xmax + xmin) / 2
	vxmaxymax := ds.ValueAt(r.Max.X-1, r.Max.Y-1)
	vxminymin := ds.ValueAt(r.Min.X-1, r.Min.Y-1)
	vxminymax := ds.ValueAt(r.Min.X-1, r.Max.Y-1)
	vxmaxymin := ds.ValueAt(r.Max.X-1, r.Min.Y-1)
	for {
		// The centre of mass is probably not a round number,
		// so we aim to iterate only to the margin of 1 pixel
		if xmax-xmin > 1 {
			vcxup := ds.ValueAt(x, r.Min.Y-1)
			vcxdown := ds.ValueAt(x, r.Max.Y-1)
			lmass := vcxdown - vcxup - vxminymax + vxminymin
			rmass := vxmaxymax - vcxdown - vxmaxymin + vcxup
			if lmass < rmass {
				xmin = x
				x = (x + xmax) / 2
			} else {
				xmax = x
				x = (x + xmin) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Note that lmass and rmass are guaranteed to be smaller than total mass
			lmass := (ds.ValueAt(xmin, r.Max.Y-1) - ds.ValueAt(xmin, r.Min.Y-1) - vxminymax + vxminymin) * 2
			rmass := (vxmaxymax - ds.ValueAt(xmax, r.Max.Y-1) - vxmaxymin + ds.ValueAt(xmax, r.Min.Y-1)) * 2
			tmass := vxmaxymax - vxmaxymin - vxminymax + vxminymin
			if (tmass - lmass) < (tmass - rmass) {
				return xmin
			} else {
				return xmax
			}
		}
	}
	return x //should never be reached
}

// Given a DSum and a Rectangle, finds y closest to line
// dividing mass in half.
// Does not do bounds checking - assumes r is inside ds.Rect
func findcy(ds *density.DSum, r image.Rectangle) int {
	ymin := r.Min.Y
	ymax := r.Max.Y
	y := (ymax + ymin) / 2
	vxmaxymax := ds.ValueAt(r.Max.X-1, r.Max.Y-1)
	vxminymin := ds.ValueAt(r.Min.X-1, r.Min.Y-1)
	vxminymax := ds.ValueAt(r.Min.X-1, r.Max.Y-1)
	vxmaxymin := ds.ValueAt(r.Max.X-1, r.Min.Y-1)
	for {
		// The centre of mass is probably not a round number,
		// so iterate to the margin of 1 pixel
		if ymax-ymin > 1 {
			vcyleft := ds.ValueAt(r.Min.X-1, y)
			vcyright := ds.ValueAt(r.Max.X-1, y)
			upmass := vcyright - vcyleft - vxmaxymin + vxminymin
			downmass := vxmaxymax - vcyright - vxminymax + vcyleft
			if upmass < downmass {
				ymin = y
				y = (y + ymax) / 2
			} else {
				ymax = y
				y = (y + ymin) / 2
			}
		} else {
			// Round down to whichever side differs the least from total mass
			// Note that lmass and rmass are guaranteed to be smaller than total mass
			upmass := ds.ValueAt(r.Max.X-1, ymin) - ds.ValueAt(r.Min.X-1, ymin) - vxmaxymin + vxminymin
			downmass := vxmaxymax - ds.ValueAt(r.Max.X-1, ymax) - vxminymax + ds.ValueAt(r.Min.X-1, ymax)
			tmass := vxmaxymax - vxmaxymin - vxminymax + vxminymin
			if (tmass - upmass) < (tmass - downmass) {
				return ymin
			} else {
				return ymax
			}
		}
	}
	return y //should never be reached
}

// Splits current cell - modifies itself to keep half of
// the current mass of the cell, returns other half as new cell
func (c *cell) Split() (child *cell) {

	child = &cell{
		North: c.North,
		South: c.South,
		Rect:  c.Rect,
		c:     0,
	}

	ncx := findcx(c.North, c.Rect)
	ncy := findcy(c.North, c.Rect)
	scx := findcx(c.South, c.Rect)
	scy := findcx(c.South, c.Rect)

	if intgr.Abs(ncx-scx) > intgr.Abs(ncy-scy) {
		// split along y axis
		child.Rect.Max.Y = (scy + ncy) / 2
		c.Rect.Min.Y = (scy + ncy) / 2
	} else if intgr.Abs(ncx-scx) < intgr.Abs(ncy-scy) || c.Rect.Dx() > c.Rect.Dy() {
		// split along x axis
		child.Rect.Max.X = (scx + ncx) / 2
		c.Rect.Min.X = (scx + ncx) / 2
	} else {
		child.Rect.Max.Y = (scy + ncy) / 2
		c.Rect.Min.Y = (scy + ncy) / 2
	}

	return
}

type splitmap struct {
	north, south *density.DSum
	cells        []*cell
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
	for i := 0; i < maxGoRoutines; i++ {
		_ = <-waitchan
	}
	return
}

func SPFrom(img image.Image) (sp *splitmap) {
	sp = new(splitmap)
	sp.north = density.DSumFrom(img, density.AvgDensity)
	sp.south = density.DSumFrom(img, density.NegAvgDensity)
	sp.cells = []*cell{&cell{
		North: sp.north,
		South: sp.south,
		Rect:  sp.north.Rect,
		c:     0}}
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
	csp.R.north = density.DSumFrom(img, density.RedDensity)
	csp.R.south = density.DSumFrom(img, density.NegRedDensity)
	csp.G.north = density.DSumFrom(img, density.GreenDensity)
	csp.G.south = density.DSumFrom(img, density.NegGreenDensity)
	csp.B.north = density.DSumFrom(img, density.BlueDensity)
	csp.B.south = density.DSumFrom(img, density.NegBlueDensity)
	csp.A.north = density.DSumFrom(img, density.AlphaDensity)
	csp.A.south = density.DSumFrom(img, density.NegAlphaDensity)

	csp.R.cells = []*cell{&cell{
		North: csp.R.north,
		South: csp.R.south,
		Rect:  csp.R.north.Rect,
		c:     0}}
	csp.G.cells = []*cell{&cell{
		North: csp.G.north,
		South: csp.G.south,
		Rect:  csp.G.north.Rect,
		c:     0}}
	csp.B.cells = []*cell{&cell{
		North: csp.B.north,
		South: csp.B.south,
		Rect:  csp.B.north.Rect,
		c:     0}}
	csp.A.cells = []*cell{&cell{
		North: csp.A.north,
		South: csp.A.south,
		Rect:  csp.A.north.Rect,
		c:     0}}
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
			for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
				for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
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
			for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
				for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
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
			for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
				for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
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
			for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
				for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
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
