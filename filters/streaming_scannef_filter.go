package filters

import (
	"context"
	"github.com/themakers/simple-crawler/crawler"
	"io"
	"text/scanner"
)

// Not going to finish it, but it's performance is reasonable, but memory consumption is near to zero
func StreamingScannerLinksFilter() crawler.FilterFunc {
	return func(ctx context.Context, r io.Reader, yieldTitle func(pos int, title string) error, yieldLink func(pos int, link string) error) error {

		var s scanner.Scanner
		s.Init(r)
		s.Filename = ""
		s.Error = func(s *scanner.Scanner, msg string) {

		}

		for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
			s.TokenText()
		}
		return nil
	}
}
