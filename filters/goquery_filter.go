package filters

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/themakers/simple-crawler/crawler"
	"io"
)

func GoQueryLinksFilter() crawler.FilterFunc {
	return func(ctx context.Context, r io.Reader, yieldTitle func(pos int, title string) error, yieldLink func(pos int, link string) error) (err error) {
		doc, err := goquery.NewDocumentFromReader(r)
		if err != nil {
			return err
		}

		doc.Filter("a").Each(func(i int, sel *goquery.Selection) {
			if val, ok := sel.Attr("href"); ok {
				if e := yieldLink(-1, val); err != nil {
					err = e
				}
			}
		})

		return
	}
}
