package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/crhym3/imgdiff"
	"github.com/opennota/screengen"
)

var (
	verbose       = flag.Bool("v", false, "Verbose")
	keepFiles     = flag.Bool("k", false, "Keep files in png format")
	n             = flag.Int("n", 5, "Number of images to compare")
	maxSameImg    = flag.Int("s", 2, "Maximal number of times 2 consecutive images can be similar")
	diffThreshold = 10.0
)

func mkdir(name string) (string, error) {
	base := name
	for i := 0; ; i++ {
		_, err := os.Stat(name)
		if os.IsNotExist(err) {
			break
		}
		name = base + fmt.Sprintf("_%d", i)
	}
	return name, os.Mkdir(name, 0755)
}

func expand(tmpl, filename string) string {
	name := path.Base(filename)
	ext := path.Ext(name)
	name = name[:len(name)-len(ext)]
	return strings.Replace(tmpl, "%n", name, -1)
}

// writeImage writes image img to the file fn.
func writeImage(img image.Image, fn string) {
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PNG encoding error: %v\n", err)
		os.Exit(1)
	}
}

func detectSpam(fn string) {
	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading video file: %v\n", err)
		os.Exit(1)
	}
	defer gen.Close()

	differ := imgdiff.NewPerceptual(2.2, 100.0, 45.0, 1.0, true)

	// Fast seek mode
	gen.Fast = true

	inc := gen.Duration / int64(*n)

	var (
		imgCur  image.Image
		imgPrev image.Image
		dname   string
	)

	dirtmpl := "%n"

	if *keepFiles {
		dname = expand(dirtmpl, fn)
		dname, err = mkdir(dname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't create directory: %v\n", err)
			os.Exit(1)
		}
	}

	imgCur = nil
	imgPrev = nil

	identicalImgCount := 0
	d := inc / 2

	for i := 0; i < *n; i++ {
		imgPrev = imgCur
		imgCur, err = gen.Image(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't generate screenshot: %v\n", err)
			os.Exit(1)
		}

		if *keepFiles {
			fntmpl := "shot%03d.png"
			fn := filepath.Join(dname, fmt.Sprintf(fntmpl, i))
			fmt.Printf("Writting %s to disk\n", fn)
			writeImage(imgCur, fn)
		}
		if imgPrev != nil && imgCur != nil {
			res, n, err := differ.Compare(imgCur, imgPrev)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Can't diff pictures")
				os.Exit(1)
			}

			np := 100.0 * (float64(n) / float64(res.Bounds().Dx()*res.Bounds().Dy()))
			if np <= diffThreshold {
				identicalImgCount++
			}
			if *verbose {
				fmt.Printf("difference: %f%%\n", np)
			}
		}
		d += inc
	}

	if identicalImgCount >= *maxSameImg {
		fmt.Printf("This is SPAM! (%d>=%d)\n", identicalImgCount, *maxSameImg)
	} else {
		fmt.Printf("This is not spam... (%d<%d)\n", identicalImgCount, *maxSameImg)
	}

}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	detectSpam(flag.Args()[0])
}
