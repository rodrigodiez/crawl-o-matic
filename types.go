package main

import (
	"fmt"
	"net/url"
	"sync"
)

type page struct {
	url   *url.URL
	links []*url.URL
}

type crawler struct {
	seedURL        *url.URL
	pages          chan *page
	errors         chan error
	urlQueue       chan *url.URL
	registeredURLs map[string]struct{}
	mutex          *sync.Mutex
}

func (c *crawler) start() {
	go func() {
		for url := range c.urlQueue {
			reader, err := fetch(url)
			if err != nil {
				c.errors <- fmt.Errorf("Unable to fetch '%s':%s", url.String(), err.Error())
			}
			links := getLinks(reader)

			for _, link := range links {
				c.register(link)
			}
		}
	}()

	c.urlQueue <- c.seedURL
}

func (c *crawler) register(link *url.URL) {
	c.mutex.Lock()

	if _, ok := c.registeredURLs[link.String()]; !ok {
		c.registeredURLs[link.String()] = struct{}{}
		c.mutex.Unlock()
		c.urlQueue <- link

		return
	}

	c.mutex.Unlock()
}
