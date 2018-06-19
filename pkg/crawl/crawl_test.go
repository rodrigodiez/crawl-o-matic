package crawl_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rodrigodiez/crawl-o-matic/pkg/crawl"
)

type pageMock struct {
	path  string
	resp  string
	links []string
}

func TestCrawlError(t *testing.T) {
	tt := []struct {
		name   string
		rawurl string
	}{
		{name: "URL is empty", rawurl: ""},
		{name: "URL is non absolute", rawurl: "/foo"},
		{name: "URL is not http/s: mail", rawurl: "mailto:rodrigo@rodrigodiez.io"},
		{name: "URL is not http/s: ftp", rawurl: "ftp://host/file.txt"},
		{name: "URL is not an URL", rawurl: "this-is-just-test"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ch, err := crawl.Crawl(tc.rawurl)

			if ch != nil {
				t.Error("channel was expected to be nil")
			}

			if err == nil {
				t.Error("error was not expected to be nil")
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

	ch, err := crawl.Crawl(hs.URL + "/foo")

	if err != nil {
		t.Error("Error was expected to be nil")
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

	ch, err := crawl.Crawl(hs.URL + "/foo")

	if err != nil {
		t.Error("Error was expected to be nil")
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
		return
	}))

	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("<a href='%s/foo'>An external link</a>", hsExternal.URL)))
	}))

	ch, err := crawl.Crawl(hs.URL)

	if err != nil {
		t.Error("Error was expected to be nil")
	}

	for range ch {
	}

	if callCount != 0 {
		t.Error("External links were not expected to be followed")
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
