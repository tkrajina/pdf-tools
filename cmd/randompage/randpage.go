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
	"strings"
	"encoding/json"
)

var rnd = rand.New(rand.NewSource(time.Now().Unix()))

func panicIfErr(err error) {
	panicIfErrf(err, "")
}

func panicIfErrf(err error, msg string, params ...interface{}) {
	if err != nil {
		panic(fmt.Sprintf(msg, params...) + ":" + err.Error())
	}
}

type Opts struct {
	OpenWith string
	MaxCount int
	NoSave bool
	Stats bool
	//VerticalHalf   bool
	//HorizontalHalf bool
}

func main() {
	var opts Opts
	flag.StringVar(&(opts.OpenWith), "open-with", "", "Open with")
	flag.IntVar(&(opts.MaxCount), "max-count", -1, "Page opened <=n times")
	flag.BoolVar(&(opts.NoSave), "no-save", false, "No save (don't update counter)")
	flag.BoolVar(&(opts.Stats), "stats", false, "Show (only) stats")

	if opts.MaxCount < 0 {
		opts.MaxCount = 1000000
	}

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

	docInfo := DocInfo{Filename: fileName}
	err = docInfo.loadDescriptor()
	panicIfErr(err)

	if opts.Stats {
		showStats(docInfo, pages)
		return
	}

	rndPage := docInfo.findRandomPage(opts, pages)
	if rndPage < 0 {
		fmt.Printf("No page for that criteria found")
		os.Exit(1)
	}
	fmt.Printf("Page %d opened %d times\n", rndPage, docInfo.getPageCounter(rndPage))

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

	docInfo.IncrementPage(rndPage)
	if opts.NoSave {
		return
	}

	err = docInfo.saveDescriptor()
	panicIfErr(err)

	showStats(docInfo, pages)
}

func showStats(info DocInfo, pages int) {
	fmt.Printf("%d pages\n", pages)

	var maxCount int
	counters := map[int]int{}
	for p := 0; p <= pages; p++ {
		count := info.getPageCounter(p)
		counters[count] ++
		if count > maxCount {
			maxCount = count
		}
	}

	for count := 0; count <= maxCount; count++ {
		p := counters[count]
		if p != 0 {
			fmt.Printf("%d (%.2f%% of total) pages are viewed %d times\n", p, float64(p)/float64(pages-1) * 100, count)
		}
	}
}

type DocInfo struct {
	Filename     string `json:"file_name"`
	PagesCounter []int  `json:"pages_counter"`
}

func (di *DocInfo) IncrementPage(page int) {
	if page >= len(di.PagesCounter) {
		newPagesCounter := make([]int, page+1)
		for n := range di.PagesCounter {
			newPagesCounter[n] = di.PagesCounter[n]
		}
		di.PagesCounter = newPagesCounter
	}
	di.PagesCounter[page] += 1
}

func (di DocInfo) descriptorFilename(pdfFilename string) string {
	return strings.Replace(pdfFilename, ".pdf", ".page.json", -1)
}

func (di *DocInfo) loadDescriptor() error {
	f, err := os.Open(di.descriptorFilename(di.Filename))
	if err != nil {
		if !os.IsExist(err) {
			di.PagesCounter = make([]int, 0)
			return nil
		}
		return err
	}

	bytes, err := ioutil.ReadAll(f)
	if err := json.Unmarshal(bytes, &di); err != nil {
		return err
	}

	return nil
}

func (di DocInfo) saveDescriptor() error {
	fn := di.descriptorFilename(di.Filename)
	fmt.Printf("Updating counters in %s\n", fn)
	f, err := os.Create(fn)
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(di, "", "    ")
	if  err != nil {
		return err
	}

	_, err = f.Write(bytes)
	return err
}

func (di *DocInfo) getPageCounter(page int) int {
	if page >= len(di.PagesCounter) {
		return 0
	}
	return di.PagesCounter[page]
}

func (di DocInfo) findRandomPage(opts Opts, pages int) int {
	pagesCandidates := make([]int, 0)
	for n := 0; n <= pages; n++ {
		count := di.getPageCounter(n)
		if count <= opts.MaxCount {
			pagesCandidates = append(pagesCandidates, n)
		} else {
			fmt.Printf("Page %d opened more than %d times (%d)\n", n, opts.MaxCount, count)
		}
	}

	if len(pagesCandidates) == 0 {
		return -1
	}

	fmt.Printf("%d/%d of pages are candidates\n", len(pagesCandidates), pages-1)

	rndPage := pagesCandidates[rnd.Int() % len(pagesCandidates)]
	fmt.Printf("Random page out of %d: %d\n", pages, rndPage)
	return rndPage
}
