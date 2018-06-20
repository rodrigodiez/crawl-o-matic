package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"time"

	"github.com/rodrigodiez/crawl-o-matic/pkg/crawl"
)

func main() {
	start := time.Now()
	count := 0

	seedURL := flag.String("url", "", "Seed URL to start crawling")
	outputPath := flag.String("output", "", "Path where to output a sitemap")
	interval := flag.Duration("interval", 1*time.Microsecond, "Interval between requests")
	maxConcurrent := flag.Int("maxConcurrent", 1, "Max number of crawling goroutines")

	flag.Parse()

	if *seedURL == "" || *outputPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Create(*outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	pages, err := crawl.Crawl(*seedURL, time.NewTicker(*interval), uint16(*maxConcurrent))
	if err != nil {
		log.Fatal(err)
	}

	for page := range pages {
		count++
		log.Printf("%s", page.URL.String())
		crawl.Write(w, *page)
	}
	elapsed := time.Since(start)
	log.Printf("Scanned %d pages in %s (%.2f/s)", count, elapsed, float64(count)/elapsed.Seconds())
}
