// Example application for voronoi/density package other 
// than voronoi diagrams. dipmap is a specialised density 
// map for creating dipoles, which can then be further
// split into dipoles. After N generations, it has
// divided a source image into 2^N cells.
package main

import (
	"code.google.com/p/go-stippling/density"
	"flag"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func main() {

	var fileName = flag.String("f", "test.jpg", "\tName of the image (f)ile to be used")
	var outputName = flag.String("o", "output", "\t\tName of the (o)utput (no extension)")
	var outputExt = flag.Uint("e", 1, "\t\tOutput (e)xtension type:\n\t\t\t 1 \t png (default)\n\t\t\t 2 \t jpg")
	var jpgQuality = flag.Int("q", 90, "\t\tJPG output (q)uality")
	var generations = flag.Uint("g", 3, "\t\tNumber of (g)enerations")
	var mono = flag.Bool("m", true, "\t\t(m)onochrome (default) or coloured output")
	var saveAll = flag.Bool("s", true, "\t\t(s)ave all generations (default) - only save last generation if false")
	var numCores = flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	flag.Parse()

	var nc int
	if *numCores <= 0 || *numCores > runtime.NumCPU() {
		nc = runtime.NumCPU()
	} else {
		nc = *numCores
	}
	runtime.GOMAXPROCS(nc)

	// Open the file.
	file, err := os.Open(*fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Decode the image.
	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	splitName := []string{}
	switch *outputExt {
	case 1:
		splitName = []string{*outputName, "00", "png"}
	case 2:
		splitName = []string{*outputName, "00", "jpg"}
	}

	toFile := func(i image.Image, g uint) {

		numSlice := []string{}

		// I highly doubt anyone would try to go beyond 99 generations,
		// as that would generate over 2^99 cells.
		if g%10 == g {
			numSlice = append(numSlice, "0")
		}
		numSlice = append(numSlice, strconv.Itoa(int(g)))
		splitName[1] = strings.Join(numSlice, "")

		output, err := os.Create(strings.Join(splitName, "."))
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
		wdm := NewWD(img, density.AvgDensity, density.NegAvgDensity, uint(1<<(*generations)))
		for i := uint(0); uint(i) < *generations; i++ {
			if *saveAll {
				wdm.Render(nc)
				toFile(wdm, i)
			}
			wdm.SplitCells(nc)
		}
		wdm.Render(nc)
		toFile(wdm, *generations)
	} else {
		cwdm := NewColWD(img, uint(1<<(*generations)))
		for i := uint(0); uint(i) < *generations; i++ {
			if *saveAll {
				cwdm.Render(nc)
				toFile(cwdm, i)
			}
			cwdm.SplitCells(nc)
		}
		cwdm.Render(nc)
		toFile(cwdm, *generations)
	}
}
