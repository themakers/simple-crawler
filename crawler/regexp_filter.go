package crawler

import (
	"context"
	"io"
	"io/ioutil"
	"regexp"
)

var filterRx = regexp.MustCompile(`<a\s+(?:[^>]*?\s+)?href="([^"]*)"`)

func RegexpLinksFilter() FilterFunc {
	return func(ctx context.Context, r io.Reader, yield func(pos int, link string) error) error {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		str := string(data)

		match := filterRx.FindAllStringSubmatchIndex(str, -1)

		for _, match := range match {
			if err := yield(match[2], str[match[2]:match[3]]); err != nil {
				return err
			}
		}

		return nil
	}
}
