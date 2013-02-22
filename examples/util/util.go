// Because all examples use the same basic structure, part of it was
// refactored into this package. Remember to import and initiate the
// flag package when using this one!
package util

import (
	"flag"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

var (
	// Flags
	outputName    *string //flag.String("o", "output", "\t\tName of the (o)utput (no extension)")
	outputExt     *uint   //flag.Uint("e", 1, "\t\tOutput (e)xtension type:\n\t\t\t 1 \t png (default)\n\t\t\t 2 \t jpg")
	jpgQuality    *int    //flag.Int("q", 90, "\t\tJPG output (q)uality")
	generations   *uint   //flag.Uint("g", 16, "\t\tNumber of (g)enerations")
	xweight       *int    //flag.Int("x", 1, "\t\tRelative weight of (x) axis")
	yweight       *int    //flag.Int("y", 1, "\t\tRelative weight of (y) axis")
	zweight       *int    //flag.Int("z", 1, "\t\tRelative weight of (z) axis")
	numCores      *int    //flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	mono          *bool   //flag.Bool("mono", true, "\t\tMonochrome or colour output")
	maxGoroutines *int

	//Exported
	Generations               uint
	Xweight, Yweight, Zweight int
	NumCores                  int
	Mono                      bool
	MaxGoroutines             int
)

func init() {
	outputName = flag.String("o", "output", "\t\tName of the (o)utput (no extension)")
	outputExt = flag.Uint("e", 1, "\t\tOutput (e)xtension type:\n\t\t\t 1 \t png (default)\n\t\t\t 2 \t jpg")
	jpgQuality = flag.Int("q", 90, "\t\tJPG output (q)uality")
	generations = flag.Uint("g", 24, "\t\tNumber of (g)enerations")
	xweight = flag.Int("x", 1, "\t\tRelative weight of (x) axis")
	yweight = flag.Int("y", 1, "\t\tRelative weight of (y) axis")
	zweight = flag.Int("z", 1, "\t\tRelative weight of (z) axis")
	numCores = flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	mono = flag.Bool("mono", true, "\t\tMonochrome or colour output")
	maxGoroutines = flag.Int("maxg", 256, "\t\tMaximum number of goroutines when splitting cells")
}

func Init() {
	if !flag.Parsed() {
		flag.Parse()
		Generations = *generations
		Xweight = *xweight
		Yweight = *yweight
		Zweight = *zweight
		NumCores = *numCores
		Mono = *mono
		if *maxGoroutines > 0 {
			MaxGoroutines = *maxGoroutines
		} else {
			MaxGoroutines = 1
		}
		if *numCores <= 0 || *numCores > runtime.NumCPU() {
			runtime.GOMAXPROCS(runtime.NumCPU())
		} else {
			runtime.GOMAXPROCS(*numCores)
		}

	}
}

func ListFiles() []string {
	files := []string{}
	list := func(filepath string, f os.FileInfo, err error) error {
		files = append(files, filepath)
		return nil
	}
	err := filepath.Walk(flag.Arg(0), list)
	if err != nil {
		log.Printf("filepath.Walk() returned %v\n", err)
	}
	return files
}

func FileToImage(fileName string) (img *image.Image, err error) {
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

func ImgToFile(img image.Image, i int) {
	output, err := os.Create(filename(i))
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	switch *outputExt {
	case 1:
		png.Encode(output, img)
	case 2:
		jpeg.Encode(output, img, &jpeg.Options{*jpgQuality})
	}
}

func filename(frameNum int) string {

	splitName := *outputName + "-" + strconv.Itoa(frameNum)
	switch *outputExt {
	case 1:
		splitName = splitName + ".png"
	case 2:
		splitName = splitName + ".jpg"
	}
	return splitName
}
