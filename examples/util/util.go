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
	generationsa  *int    //flag.Int("ga", -1, "\t\tNumber of alpha generations - superseded by -g if negative")
	saveAll       *bool   //flag.Bool("s", true, "\t\t(s)ave all generations (default) - only save last generation if false")
	xweight       *int    //flag.Int("x", 1, "\t\tRelative weight of (x) axis")
	yweight       *int    //flag.Int("y", 1, "\t\tRelative weight of (y) axis")
	zweight       *int    //flag.Int("z", 1, "\t\tRelative weight of (z) axis")
	numCores      *int    //flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	mono          *bool   //flag.Bool("mono", true, "\t\tMonochrome or colour output")
	maxGoroutines *int
	verbose       *bool

	//Exported
	Generations                int
	GenerationsR, GenerationsG int
	GenerationsB, GenerationsA int
	SaveAll, Verbose           bool
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
	generationsa = flag.Int("ga", 0, "\t\tNumber of blue generations - superseded by -g if negative")
	saveAll = flag.Bool("s", true, "\t\t(s)ave all generations (default) - only save last generation if false")
	xweight = flag.Int("x", 1, "\t\tRelative weight of (x) axis")
	yweight = flag.Int("y", 1, "\t\tRelative weight of (y) axis")
	zweight = flag.Int("z", 1, "\t\tRelative weight of (z) axis")
	numCores = flag.Int("c", 1, "\t\tMax number of (c)ores to be used.\n\t\t\tUse all available cores if less or equal to zero")
	mono = flag.Bool("mono", true, "\t\tMonochrome or colour output")
	verbose = flag.Bool("v", false, "\t\tVerbose output")
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
	if GenerationsA = *generationsa; GenerationsA < 0 {
		GenerationsA = Generations
	}
	Generations = intgr.Max(intgr.Max(intgr.Max(Generations, GenerationsA), intgr.Max(GenerationsR, GenerationsG)), GenerationsB)
	SaveAll = *saveAll
	Verbose = *verbose
	Xweight = intgr.Max(0, *xweight)
	Yweight = intgr.Max(0, *yweight)
	Zweight = intgr.Max(0, *zweight)
	Mono = *mono
	if *maxGoroutines > 0 {
		MaxGoroutines = *maxGoroutines
	} else {
		MaxGoroutines = 1
	}
	if *numCores <= 0 || *numCores > runtime.NumCPU() {
		NumCores = runtime.NumCPU()
	} else {
		NumCores = *numCores
	}
	runtime.GOMAXPROCS(NumCores)

}

func ListFiles() [][]string {
	argList := flag.Args()
	filesList := make([][]string, len(argList))
	for i, arg := range argList {
		files := []string{}
		list := func(filepath string, f os.FileInfo, err error) error {
			files = append(files, filepath)
			return nil
		}
		err := filepath.Walk(arg, list)
		if err != nil && Verbose {
			log.Printf("filepath.Walk() returned %v\n", err)
		}
		filesList[i] = files
	}
	return filesList
}

func FileToImage(fileName string) (img *image.Image, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		if Verbose {
			log.Println(err)
		}
		return nil, err
	}
	defer file.Close()

	image, _, err := image.Decode(file)
	if err != nil && Verbose {
		log.Println(err, "Could not decode image:", fileName)
	}
	return &image, err
}

func ImgToFile(img image.Image, i ...int) {
	output, err := os.Create(filename(i...))
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

func filename(i ...int) string {

	splitName := *outputName
	for _, s := range i {
		splitName = splitName + "-" + strconv.Itoa(s)
	}
	switch *outputExt {
	case 1:
		splitName = splitName + ".png"
	case 2:
		splitName = splitName + ".jpg"
	}
	return splitName
}
