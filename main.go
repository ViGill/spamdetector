package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViGill/imgdiff"
	"github.com/cheggaaa/pb"
	"github.com/opennota/screengen"
)

var (
	verbose       = flag.Bool("v", false, "Verbose")
	progressBar   = flag.Bool("p", false, "Progress bar")
	keepFiles     = flag.Bool("k", false, "Keep files in png format")
	n             = flag.Int("n", 5, "Number of images to compare")
	maxSameImg    = flag.Int("s", 2, "Maximal number of times 2 consecutive images can be similar")
	diffThreshold = 1.0
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
		os.Exit(0)
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PNG encoding error: %v\n", err)
		os.Exit(0)
	}
}

func detectSpam(fn string) bool {
	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading video file: %v\n", err)
		return false
	}
	defer gen.Close()

	//	differ := imgdiff.NewPerceptual(2.2, 100.0, 45.0, 1.0, true)
	differ := imgdiff.NewBinary()

	// Fast seek mode
	//	gen.Fast = true

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
			os.Exit(0)
		}
	}

	imgCur = nil
	imgPrev = nil

	identicalImgCount := 0
	d := inc / 2

	var bar *pb.ProgressBar

	if *progressBar {
		bar = pb.StartNew(*n)
	}

	for i := 0; i < *n; i++ {
		imgPrev = imgCur
		imgCur, err = gen.Image(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't generate screenshot: %v\n", err)
			return false
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
				return false
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
		if *progressBar {
			bar.Increment()
		}
	}

	if *progressBar {
		bar.Finish()
	}

	if identicalImgCount >= *maxSameImg {
		fmt.Printf("%s is SPAM\n", fn)
		return true
	} else {
		fmt.Printf("%s is safe\n", fn)
		return false
	}

}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	fi, err := os.Stat(flag.Args()[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if fi.Mode().IsDir() {
		files, _ := ioutil.ReadDir(flag.Args()[0])
		for _, f := range files {
			fi2, err := os.Stat(f.Name())
			if err != nil || fi2.Mode().IsDir() {
				continue
			}
			if *verbose {
				fmt.Fprintf(os.Stderr, "Checking %s\n", f.Name())
			}
			detectSpam(flag.Args()[0] + "/" + f.Name())
		}
	} else {
		if detectSpam(flag.Args()[0]) {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}
