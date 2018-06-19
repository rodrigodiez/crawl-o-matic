package main

import (
	"bufio"
	"flag"
	"fmt"
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

	pages, err := crawl.Crawl(*seedURL)
	if err != nil {
		log.Fatal(err)
	}

	for page := range pages {
		count++
		log.Printf("%s", page.URL.String())
		fmt.Fprintf(w, "[%s]\n", page.URL.String())

		for _, link := range page.Links {
			fmt.Fprintf(w, "%s\n", link.String())
		}

		fmt.Fprintf(w, "\n")
	}
	log.Printf("Scanned %d pages in %s", count, time.Since(start))
}
