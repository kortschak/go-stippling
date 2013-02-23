// Because all examples use the same basic structure, part of it was
// refactored into this package. Remember to import and initiate the
// flag package when using this one!
package util

import (
	"code.google.com/p/intmath/intgr"
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
	generations   *int    //flag.Int("g", 16, "\t\tNumber of (g)enerations")
	generationsr  *int    //flag.Int("gr", -1, "\t\tNumber of red generations - superseded by -g if negative")
	generationsg  *int    //flag.Int("gg", -1, "\t\tNumber of green generations - superseded by -g if negative")
	generationsb  *int    //flag.Int("gb", -1, "\t\tNumber of blue generations - superseded by -g if negative")
	xweight       *int    //flag.Int("x", 1, "\t\tRelative weight of (x) axis")
	yweight       *int    //flag.Int("y", 1, "\t\tRelative weight of (y) axis")
	zweight       *int    //flag.Int("z", 1, "\t\tRelative weight of (z) axis")
	numCores      *int    //flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	mono          *bool   //flag.Bool("mono", true, "\t\tMonochrome or colour output")
	maxGoroutines *int

	//Exported
	Generations, GenerationsR  int
	GenerationsG, GenerationsB int
	Xweight, Yweight, Zweight  int
	NumCores                   int
	Mono                       bool
	MaxGoroutines              int
)

func init() {
	outputName = flag.String("o", "output", "\t\tName of the (o)utput (no extension)")
	outputExt = flag.Uint("e", 1, "\t\tOutput (e)xtension type:\n\t\t\t 1 \t png (default)\n\t\t\t 2 \t jpg")
	jpgQuality = flag.Int("q", 90, "\t\tJPG output (q)uality")
	generations = flag.Int("g", 24, "\t\tNumber of (g)enerations")
	generationsr = flag.Int("gr", -1, "\t\tNumber of red generations - superseded by -g if negative")
	generationsg = flag.Int("gg", -1, "\t\tNumber of green generations - superseded by -g if negative")
	generationsb = flag.Int("gb", -1, "\t\tNumber of blue generations - superseded by -g if negative")
	xweight = flag.Int("x", 1, "\t\tRelative weight of (x) axis")
	yweight = flag.Int("y", 1, "\t\tRelative weight of (y) axis")
	zweight = flag.Int("z", 1, "\t\tRelative weight of (z) axis")
	numCores = flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	mono = flag.Bool("mono", true, "\t\tMonochrome or colour output")
	maxGoroutines = flag.Int("mg", 256, "\t\tMaximum number of goroutines when splitting cells")
}

func Init() {
	if !flag.Parsed() {
		flag.Parse()
	}
	Generations = *generations
	if GenerationsR = *generationsr; GenerationsR < 0 {
		GenerationsR = Generations
	}
	if GenerationsG = *generationsg; GenerationsG < 0 {
		GenerationsG = Generations
	}
	if GenerationsB = *generationsb; GenerationsB < 0 {
		GenerationsB = Generations
	}
	Xweight = intgr.Max(0, *xweight)
	Yweight = intgr.Max(0, *yweight)
	Zweight = intgr.Max(0, *zweight)
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
