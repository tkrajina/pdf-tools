package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"time"

	unipdf "github.com/unidoc/unidoc/pdf"
)

var rnd = rand.New(rand.NewSource(time.Now().Unix()))

func panicIfErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

type Opts struct {
	OpenWith string
	//VerticalHalf   bool
	//HorizontalHalf bool
}

func main() {
	var opts Opts
	flag.StringVar(&(opts.OpenWith), "open-with", "", "Open with")
	//opts.VerticalHalf = flag.Bool("vertical", "", "Random vertical half")
	//opts.HorizontalHalf = flag.Bool("horizontal", "", "Random horizontal half")
	flag.Parse()
	files := flag.Args()

	var inFile, outFile string
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No input files")
		os.Exit(1)
	} else if len(files) == 1 {
		inFile = files[0]
		if len(opts.OpenWith) > 0 {
			f, err := ioutil.TempFile("", "pdfrandompage")
			panicIfErr(err)
			f.Close()
			outFile = f.Name() + ".pdf"
		} else {
			outFile = fmt.Sprintf("rnd-page-%s", inFile)
		}
	} else if len(files) == 2 {
		inFile = files[0]
		outFile = files[1]
	} else {
		fmt.Fprintf(os.Stderr, "Too many filenames")
		os.Exit(1)
	}

	openRandomPage(inFile, outFile, opts)
}

func openRandomPage(fileName, outFile string, opts Opts) {
	fmt.Printf("Reading %s...\n", fileName)
	f, err := os.Open(fileName)
	panicIfErr(err)

	reader, err := unipdf.NewPdfReader(f)
	panicIfErr(err)

	pages, err := reader.GetNumPages()
	panicIfErr(err)

	if pages <= 0 {
		fmt.Fprintf(os.Stderr, "Zero pages")
		os.Exit(1)
	}

	rndPage := rnd.Int() % pages
	fmt.Printf("Random page out of %d: %d\n", pages, rndPage)

	writer := unipdf.NewPdfWriter()

	page, err := reader.GetPageAsPdfPage(rndPage)
	panicIfErr(err)

	pageObj := page.GetPageAsIndirectObject()

	err = writer.AddPage(pageObj)
	panicIfErr(err)

	f.Close()

	fmt.Printf("Saving to %s...\n", outFile)
	f, err = os.Create(outFile)
	panicIfErr(err)

	writer.Write(f)
	f.Close()

	if len(opts.OpenWith) > 0 {
		fmt.Printf("Opening %s with %s\n", outFile, opts.OpenWith)
		cmd := exec.Command(opts.OpenWith, outFile)
		resp, err := cmd.CombinedOutput()
		fmt.Println(string(resp))
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
}
