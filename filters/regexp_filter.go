package filters

import (
	"context"
	"github.com/themakers/simple-crawler/crawler"
	"io"
	"io/ioutil"
	"regexp"
)

var filterRx = regexp.MustCompile(`<a\s+(?:[^>]*?\s+)?href="([^"]*)"`)

// Do not need to implement streaming capabilities in RegExp filter, because it does not worth it, it's not performant.
func RegexpLinksFilter() crawler.FilterFunc {
	return func(ctx context.Context, r io.Reader, yieldTitle func(pos int, title string) error, yieldLink func(pos int, link string) error) error {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		str := string(data)

		match := filterRx.FindAllStringSubmatchIndex(str, -1)

		for _, match := range match {
			if err := yieldLink(match[2], str[match[2]:match[3]]); err != nil {
				return err
			}
		}

		return nil
	}
}
