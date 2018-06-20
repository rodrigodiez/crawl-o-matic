package crawl_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/rodrigodiez/crawl-o-matic/pkg/crawl"
)

type pageMock struct {
	path  string
	resp  string
	links []string
}

func TestCrawlError(t *testing.T) {
	tt := []struct {
		name          string
		rawurl        string
		ticker        *time.Ticker
		maxConcurrent uint16
	}{
		{name: "URL is empty", rawurl: "", ticker: time.NewTicker(1 * time.Microsecond), maxConcurrent: 10},
		{name: "URL is non absolute", rawurl: "/foo", ticker: time.NewTicker(1 * time.Microsecond), maxConcurrent: 10},
		{name: "URL is not http/s: mail", rawurl: "mailto:rodrigo@rodrigodiez.io", ticker: time.NewTicker(1 * time.Microsecond), maxConcurrent: 10},
		{name: "URL is not http/s: ftp", rawurl: "ftp://host/file.txt", ticker: time.NewTicker(1 * time.Microsecond), maxConcurrent: 10},
		{name: "URL is not an URL", rawurl: "seriously@malformed://url", ticker: time.NewTicker(1 * time.Microsecond), maxConcurrent: 10},
		{name: "Ticker is nil", rawurl: "https://www.monzo.com", ticker: nil, maxConcurrent: 10},
		{name: "maxConcurrent is 0", rawurl: "https://www.monzo.com", ticker: time.NewTicker(1 * time.Microsecond), maxConcurrent: 0},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ch, err := crawl.Crawl(tc.rawurl, tc.ticker, tc.maxConcurrent)

			if ch != nil {
				t.Error("channel should be nil")
			}

			if err == nil {
				t.Error("error should not be nil")
			}
		})
	}

}

func TestCrawl(t *testing.T) {
	mocks := []pageMock{
		{path: "/foo", resp: "<a href='/bar'>A link</a>", links: []string{"/bar"}},
		{path: "/bar", resp: "<a href='/baz'>Another link</a><a href='/qux'>Yet another link</a>", links: []string{"/baz", "/qux"}},
		{path: "/baz", resp: "<a href='https://wwww.google.com'>An external link</a>", links: []string{"https://wwww.google.com"}},
		{path: "/qux", resp: "<a href='/foo'>A link</a><a href='/foo'>A duplicated link</a>", links: []string{"/foo"}},
	}

	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock, ok := findMockByURL(mocks, r.RequestURI)

		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		w.Write([]byte(mock.resp))
	}))

	ch, err := crawl.Crawl(hs.URL+"/foo", time.NewTicker(1*time.Microsecond), 10)

	if err != nil {
		t.Error("Error should be nil")
	}

	pages := []*crawl.Page{}

	for page := range ch {
		pages = append(pages, page)
	}

	if len(pages) != len(mocks) {
		t.Errorf("%d pages were expected but got %d from the channel\n", len(mocks), len(pages))
	}

	for _, mock := range mocks {
		page, ok := findPageByURL(pages, hs.URL+mock.path)

		if !ok {
			t.Errorf("%s was expected to be returned as a page but it was not", mock.path)
			continue
		}

		if len(mock.links) != len(page.Links) {
			t.Errorf("%d links were expected but %d were found on %s", len(mock.links), len(page.Links), mock.path)
		}

		for _, linkRawURL := range mock.links {
			linkURL, _ := url.Parse(linkRawURL)
			if !linkURL.IsAbs() {
				linkURL, _ = url.Parse(hs.URL + linkRawURL)
			}

			if !pageContainsLink(page, linkURL.String()) {
				t.Errorf("%s was expected to be found as a link on %s but was not", linkURL.String(), mock.path)
			}
		}
	}
}

func TestCrawlVisitsPagesOnlyOnce(t *testing.T) {
	var callCount int
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("<a href='/foo'>A self link</a><a href='/foo'>Another self link</a>"))
	}))

	ch, err := crawl.Crawl(hs.URL+"/foo", time.NewTicker(1*time.Microsecond), 10)

	if err != nil {
		t.Error("Error should be nil")
	}

	for range ch {
	}

	if callCount != 1 {
		t.Errorf("1 calls to the server were expected but %d were made\n", callCount)
	}
}

func TestCrawlDoesNotFollowExternalLinks(t *testing.T) {
	var callCount int

	hsExternal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		http.Error(w, "Not found", http.StatusNotFound)
	}))

	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("<a href='%s/foo'>An external link</a>", hsExternal.URL)))
	}))

	ch, err := crawl.Crawl(hs.URL, time.NewTicker(1*time.Microsecond), 10)

	if err != nil {
		t.Error("Error should be nil")
	}

	for range ch {
	}

	if callCount != 0 {
		t.Error("External links should not be followed")
	}
}

func TestCrawlIgnoresMalformedLinks(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<a href='seriously@malformed://url'>An external link</a>"))
	}))

	ch, err := crawl.Crawl(hs.URL, time.NewTicker(1*time.Microsecond), 10)

	if err != nil {
		t.Error("Error should be nil")
	}

	page := <-ch

	if len(page.Links) != 0 {
		t.Error("Malformed links should have been ignored")
	}
}

func TestCrawlIgnoresHttpErrorPages(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusPermanentRedirect)
	}))

	ch, err := crawl.Crawl(hs.URL, time.NewTicker(1*time.Microsecond), 10)

	if err != nil {
		t.Error("Error should be nil")
	}

	pages := []*crawl.Page{}

	for page := range ch {
		pages = append(pages, page)
	}

	if len(pages) != 0 {
		t.Errorf("Http errors should be ignored but %d error pages were returned\n", len(pages))
	}
}

func findPageByURL(pages []*crawl.Page, url string) (*crawl.Page, bool) {
	for _, page := range pages {
		if page.URL.String() == url {
			return page, true
		}
	}

	return nil, false
}

func findMockByURL(mocks []pageMock, url string) (*pageMock, bool) {
	for _, mock := range mocks {
		if mock.path == url {
			return &mock, true
		}
	}

	return nil, false
}

func pageContainsLink(page *crawl.Page, url string) bool {
	for _, linkURL := range page.Links {
		if linkURL.String() == url {
			return true
		}
	}

	return false
}
