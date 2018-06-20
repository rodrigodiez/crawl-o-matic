package crawl

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/html"
)

// Page represents a web page. It contains its URL and a list of URLs that page links to
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
	ticker        *time.Ticker
}

// Crawl visits the given URL and follows its links within the same domain.
// The following rules apply
// - It will only follow links within the same domain of the initial URL
// - It will only follow http/https links
// - It will only visit the same link once, even if found in multiple pages
// - It will add all valid links to the list, regardless if the link is within the same domain or not
// - A page will not be added if visiting it resulted in a http.Get error (too many redirects, protocol error)
// - It will crawl at the pace that the ticker sets to avoid generating problems to crawled sites or being banned by them
// - It will use concurrency to crawl, but maxConcurrent will be respected
func Crawl(rawurl string, ticker *time.Ticker, maxConcurrent uint16) (<-chan *Page, error) {

	if maxConcurrent == 0 {
		return nil, errors.New("maxConcurrent must be greater than 0")
	}

	if ticker == nil {
		return nil, errors.New("ticker must not be nil")
	}

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
		queue:         make(chan *url.URL, maxConcurrent),
		registry:      make(map[string]struct{}),
		registryMutex: &sync.Mutex{},
		pages:         make(chan *Page),
		ticker:        ticker,
	}

	crawler.start()

	return crawler.pages, nil
}

func (c *crawler) start() {

	c.register(c.seed)

	go func() {
		for range c.ticker.C {
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

	if link.Host != c.seed.Host {
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
