package crawler

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"io"
)

func GoQueryLinksFilter() FilterFunc {
	return func(ctx context.Context, r io.Reader, yield func(pos int, link string) error) (err error) {
		doc, err := goquery.NewDocumentFromReader(r)
		if err != nil {
			return err
		}

		doc.Filter("a").Each(func(i int, sel *goquery.Selection) {
			if val, ok := sel.Attr("href"); ok {
				if e := yield(-1, val); err != nil {
					err = e
				}
			}
		})

		return
	}
}
