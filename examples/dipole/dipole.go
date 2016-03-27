// Example application for voronoi/density package other
// than voronoi diagrams. dipmap is a specialised density
// map for creating dipoles, which can then be further
// split into dipoles. After N generations, it has
// divided a source image into 2^N cells.
package main

import (
	"flag"
	"github.com/kortschak/go-stippling/density"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

func main() {

	var outputName = flag.String("o", "output", "\t\tName of the (o)utput (no extension)")
	var outputExt = flag.Uint("e", 1, "\t\tOutput (e)xtension type:\n\t\t\t 1 \t png (default)\n\t\t\t 2 \t jpg")
	var jpgQuality = flag.Int("q", 90, "\t\tJPG output (q)uality")
	var generations = flag.Uint("g", 3, "\t\tNumber of (g)enerations")
	var mono = flag.Bool("m", true, "\t\t(m)onochrome (default) or coloured output")
	var saveAll = flag.Bool("s", true, "\t\t(s)ave all generations (default) - only save last generation if false")
	var numCores = flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	flag.Parse()

	var nc int //used later as well, in case you're wondering
	if *numCores <= 0 || *numCores > runtime.NumCPU() {
		nc = runtime.NumCPU()
	} else {
		nc = *numCores
	}
	runtime.GOMAXPROCS(nc)

	// Use a function variable for processing the files, so that defer
	// gets called for closing the fields, but we don't have to pass
	// all of the variables. This feels dirty way of doing this, but
	// it works.
	var fileNum int
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
		toFile := func(i image.Image, g uint) {

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

			output, err := os.Create(splitName)
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
			dm := NewDMap(img, density.AvgDensity, density.NegAvgDensity, uint(1<<(*generations)))
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					dm.Render(nc)
					toFile(dm, i)
				}
				dm.SplitCells(nc)
			}
			dm.Render(nc)
			toFile(dm, *generations)
		} else {
			cdm := NewColDMap(img, uint(1<<(*generations)))
			for i := uint(0); uint(i) < *generations; i++ {
				if *saveAll {
					cdm.Render(nc)
					toFile(cdm, i)
				}
				cdm.SplitCells(nc)
			}
			cdm.Render(nc)
			toFile(cdm, *generations)
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
