package main

import (
	"context"
	"fmt"
	"net/url"
)

func main() {

}

func crawl(ctx context.Context, seedURL *url.URL) (*crawler, error) {

	if !seedURL.IsAbs() {
		return nil, fmt.Errorf("crawl::Not an absolute URL:%s", seedURL.String())
	}

	if seedURL.Scheme != "http" && seedURL.Scheme != "https" {
		return nil, fmt.Errorf("crawl::Not an http(s) URL:%s", seedURL.String())
	}

	crawler := &crawler{
		seedURL:     seedURL,
		urlQueue:    make(chan *url.URL),
		visitedUrls: make(map[string]struct{}),
		pages:       make(chan *page),
		errors:      make(chan error),
	}

	crawler.start()

	return crawler, nil
}
