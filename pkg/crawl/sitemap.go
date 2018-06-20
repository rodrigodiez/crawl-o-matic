package crawl

import (
	"fmt"
	"io"
)

// Write will write page into w with a default format
func Write(w io.Writer, page Page) {
	fmt.Fprintf(w, "[%s]\n", page.URL.String())

	for _, link := range page.Links {
		fmt.Fprintf(w, "%s\n", link.String())
	}

	fmt.Fprintf(w, "\n")
}
