package crawler

import (
	"context"
	"golang.org/x/net/html"
	"io"
)

func StreamingGoHTMLLinksFilter() FilterFunc {
	return func(ctx context.Context, r io.Reader, yield func(pos int, link string) error) error {
		getHref := func(t html.Token) (ok bool, href string) {
			for _, a := range t.Attr {
				if a.Key == "href" {
					href = a.Val
					ok = true
				}
			}
			return
		}

		z := html.NewTokenizer(r)

		for {
			tt := z.Next()

			switch {
			case tt == html.ErrorToken:
				// End of the document, we're done
				return nil
			case tt == html.StartTagToken:
				t := z.Token()

				// Check if the token is an <a> tag
				isAnchor := t.Data == "a"
				if !isAnchor {
					continue
				}

				// Extract the href value, if there is one
				ok, url := getHref(t)
				if !ok {
					continue
				}

				if err := yield(-1, url); err != nil {
					return err
				}
			}
		}
	}
}
