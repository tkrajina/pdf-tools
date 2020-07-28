package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	unipdf "github.com/unidoc/unidoc/pdf"
)

func main() {
	crop := flag.String("crop", "", "Pages from-to")
	flag.Parse()

	inFile := flag.Arg(0)
	outFile := flag.Arg(1)

	var opts PdfOpts

	if len(*crop) > 0 {
		_, crops, err := parseNumbers(*crop)
		fmt.Printf("crops=%#v\n", crops)
		if err != nil {
			printDefaultsAndExit(err.Error())
		}
		if len(crops) == 2 {
			opts.percentageTop = crops[0]
			opts.percentageBottom = crops[0]
			opts.percentageLeft = crops[1]
			opts.percentageRight = crops[1]
		} else if len(crops) == 4 {
			opts.percentageTop = crops[0]
			opts.percentageRight = crops[1]
			opts.percentageBottom = crops[2]
			opts.percentageLeft = crops[3]
		} else {
			printDefaultsAndExit("Invalid crop percentages")
		}
	}

	fmt.Printf("opts=%#v\n", opts)
	if err := splitPdf(inFile, outFile, opts); err != nil {
		panic(err.Error())
	}
}

func parseNumbers(str string) ([]int, []float64, error) {
	if str == "" {
		return []int{}, []float64{}, nil
	}
	delimiter := "-"
	if !strings.Contains(str, delimiter) {
		delimiter = ","
	}
	parts := strings.Split(str, delimiter)
	resInts := make([]int, len(parts))
	resFloats := make([]float64, len(parts))
	for n := range parts {
		i, err := strconv.ParseInt(parts[n], 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot parse number '%s' in '%s'", parts[n], str)
		}
		resInts[n] = int(i)
		f, err := strconv.ParseFloat(parts[n], 64)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot parse number '%s' in '%s'", parts[n], str)
		}
		resFloats[n] = f
	}
	return resInts, resFloats, nil
}

func printDefaultsAndExit(msg string) {
	fmt.Fprintf(os.Stderr, msg+"\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func panicIfErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

type PdfOpts struct {
	percentageTop, percentageBottom float64
	percentageLeft, percentageRight float64
}

func splitPdf(inputPath string, outputPath string, opts PdfOpts) error {
	pdfWriter := unipdf.NewPdfWriter()

	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("Error openning %s: %s", inputPath, err.Error())
	}

	defer f.Close()

	pdfReader, err := unipdf.NewPdfReader(f)
	if err != nil {
		return err
	}

	isEncrypted, err := pdfReader.IsEncrypted()
	if err != nil {
		return err
	}

	if isEncrypted {
		_, err = pdfReader.Decrypt([]byte(""))
		if err != nil {
			return err
		}
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return err
	}

	for pageNum := 1; pageNum <= numPages; pageNum++ {

		page, err := pdfReader.GetPageAsPdfPage(pageNum)
		if err != nil {
			return err
		}

		bbox, err := page.GetMediaBox()
		if err != nil {
			return err
		}

		// Zoom in on the page middle, with a scaled width and height.
		width := (*bbox).Urx - (*bbox).Llx
		height := (*bbox).Ury - (*bbox).Lly
		_ = width
		_ = height
		(*bbox).Llx += width * (float64(opts.percentageLeft) / 100.)
		(*bbox).Lly += height * (float64(opts.percentageBottom) / 100.)
		(*bbox).Urx -= width * (float64(opts.percentageRight) / 100.)
		(*bbox).Ury -= height * (float64(opts.percentageTop) / 100.)
		page.MediaBox = bbox

		pageObj := page.GetPageAsIndirectObject()

		err = pdfWriter.AddPage(pageObj)
		if err != nil {
			return err
		}
	}

	fWrite, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	defer fWrite.Close()

	err = pdfWriter.Write(fWrite)
	if err != nil {
		return err
	}

	return nil
}
