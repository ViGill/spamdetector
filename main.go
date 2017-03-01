package main

import (
	"flag"
	"fmt"
	"github.com/crhym3/imgdiff"
	"github.com/opennota/screengen"
	"image"
	"os"
)

var (
	n             = flag.Int("n", 5, "Number of images to compare")
	maxSameImg    = flag.Int("s", 2, "Number of similar images to report as spam")
	diffThreshold = 10.0
)

func detectSpam(fn string) {
	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading video file: %v\n", err)
		os.Exit(1)
	}
	defer gen.Close()

	gen.Fast = true

	inc := gen.Duration / int64(*n)
	differ := imgdiff.NewBinary()

	var (
		imgCur  image.Image
		imgPrev image.Image
	)

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

			//			fmt.Printf("difference: %f%%\n", np)
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
