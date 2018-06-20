package crawl_test

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/rodrigodiez/crawl-o-matic/pkg/crawl"
)

func TestPrintPage(t *testing.T) {
	URL, _ := url.Parse("https://www.monzo.com")
	aboutLink, _ := url.Parse("https://www.monzo.com/about")
	careersLink, _ := url.Parse("https://www.monzo.com/careers")

	page := &crawl.Page{
		URL:   URL,
		Links: []*url.URL{aboutLink, careersLink},
	}

	f := &bytes.Buffer{}

	crawl.Write(f, *page)

	expected := "[https://www.monzo.com]\nhttps://www.monzo.com/about\nhttps://www.monzo.com/careers\n\n"

	if f.String() != expected {
		t.Errorf("Sitemap was not written as expected. Got:\n%s\nExpected:\n%s", f.String(), expected)
	}
}
