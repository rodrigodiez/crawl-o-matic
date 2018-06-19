package crawl

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"golang.org/x/net/html"
)

type Page struct {
	URL   *url.URL
	Links []*url.URL
}

type crawler struct {
	seed          *url.URL
	queue         chan *url.URL
	inProgress    int64
	registry      map[string]struct{}
	registryMutex *sync.Mutex
	pages         chan *Page
}

func Crawl(rawurl string) (<-chan *Page, error) {

	seedURL, err := url.Parse(rawurl)

	if err != nil {
		return nil, err
	}

	if !seedURL.IsAbs() {
		return nil, fmt.Errorf("Not an absolute URL:%s", seedURL.String())
	}

	if seedURL.Scheme != "http" && seedURL.Scheme != "https" {
		return nil, fmt.Errorf("Not an http(s) URL:%s", seedURL.String())
	}

	crawler := &crawler{
		seed:          seedURL,
		queue:         make(chan *url.URL, 20),
		registry:      make(map[string]struct{}),
		registryMutex: &sync.Mutex{},
		pages:         make(chan *Page),
	}

	crawler.start()

	return crawler.pages, nil
}

func (c *crawler) start() {

	c.register(c.seed)

	go func() {
		for {
			select {
			case URL := <-c.queue:
				atomic.AddInt64(&c.inProgress, 1)
				go c.visit(URL)
			default:
				if atomic.LoadInt64(&c.inProgress) == 0 {
					close(c.pages)
					return
				}
			}
		}
	}()

}

func (c *crawler) visit(link *url.URL) {
	defer atomic.AddInt64(&c.inProgress, -1)

	resp, err := http.Get(link.String())

	if err != nil {
		return
	}

	defer resp.Body.Close()

	links := c.extractLinks(resp.Body, link)

	for _, link := range links {
		c.register(link)
	}

	c.pages <- &Page{
		URL:   link,
		Links: links,
	}
}

func (c *crawler) extractLinks(reader io.Reader, baseURL *url.URL) []*url.URL {
	var links = []*url.URL{}
	var linksMap = make(map[string]*url.URL)

	tokenizer := html.NewTokenizer(reader)

tokens:
	for tt := tokenizer.Next(); tt != html.ErrorToken; tt = tokenizer.Next() {

		if tt == html.StartTagToken {
			t := tokenizer.Token()

			if t.Data == "a" {
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						linkURL, err := url.Parse(attr.Val)

						if err != nil {
							continue tokens
						}

						if !linkURL.IsAbs() {
							linkURL.Scheme = baseURL.Scheme
							linkURL.Host = baseURL.Host
							linkURL.User = baseURL.User
						}

						if linkURL.Scheme == "http" || linkURL.Scheme == "https" {
							linksMap[linkURL.String()] = linkURL
						}
						continue tokens
					}
				}
			}
		}
	}

	for _, link := range linksMap {
		links = append(links, link)
	}

	return links
}

func (c *crawler) register(link *url.URL) {

	if link.Hostname() != c.seed.Hostname() {
		return
	}

	c.registryMutex.Lock()
	if _, ok := c.registry[link.String()]; !ok {
		c.registry[link.String()] = struct{}{}
		c.registryMutex.Unlock()
		c.queue <- link

		return
	}

	c.registryMutex.Unlock()
}
