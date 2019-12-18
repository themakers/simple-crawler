package crawler

import (
	"context"
	"io"
	"strings"
)

// FIXME Strange behaviour on larger chunks
// FIXME Strange behaviour on smaller chunks
// TODO Make it fetch 'href's only from 'a' tags
func StreamingLinksFilter(chunkSize int) FilterFunc {
	return func(ctx context.Context, r io.Reader, yield func(pos int, link string) error) error {
		if chunkSize < 1024 {
			chunkSize = 1024
		}

		const (
			hrefStart = "href"
		)

		var (
			buf            = make([]byte, 0, 32*1024) //> Average page size on the internet, IMO
			str            = ""
			readTotal      = 0
			hrefMarkEndPos = -1
			hrefValuePos   = -1
			eof            = false
		)

		readMore := func() error {
			n, err := r.Read(buf[len(buf) : len(buf)+chunkSize])
			if err != nil && err != io.EOF {
				return err
			}
			buf = buf[:len(buf)+n]
			readTotal += n

			if err == io.EOF {
				eof = true
			}

			str = string(buf) //> []byte->string conversions is almost no-cost since go1.5

			return nil
		}

		dropHeadUntil := func(i int) {
			//> GC will eat it

			origLen := len(buf)

			buf = buf[i:]
			str = string(buf) //> []byte->string conversions is almost no-cost since go1.5

			newLen := len(buf)

			delta := origLen - newLen

			hrefMarkEndPos -= delta
			hrefValuePos -= delta
		}

		ensureCapacity := func() {
			if cap(buf)-len(buf) < chunkSize {
				tmp := buf
				buf = make([]byte, len(tmp), len(tmp)+chunkSize)
				copy(buf, tmp)
			}
		}

		findHrefMarkEnd := func() bool {
			if i := strings.Index(str, hrefStart); i >= 0 {
				hrefMarkEndPos = i + len(hrefStart)
				return true
			} else {
				return false
			}
		}

		findHrefValueStart := func() bool {
			if strings.HasPrefix(str, "=\"") {
				hrefValuePos = 2
				return true
			} else if strings.HasPrefix(str, "='") {
				hrefValuePos = 2
				return true
			} else if strings.HasPrefix(str, "=") {
				hrefValuePos = 1
				return true
			} else {
				return false
			}
		}

		findHrefValueEnd := func() int {
			return strings.IndexAny(str, " \"'")
		}

		for {

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			ensureCapacity()

			if err := readMore(); err != nil {
				return err
			}

			for {
				if hrefValuePos >= 0 {

					if end := findHrefValueEnd(); end >= 0 {
						if err := yield(readTotal-len(str)+hrefValuePos, str[:end]); err != nil {
							return err
						}

						dropHeadUntil(end + 1)

						continue
					} else {
						break
					}

				} else if hrefMarkEndPos >= 0 {

					if findHrefValueStart() {
						dropHeadUntil(hrefValuePos)
						continue
					} else {
						break
					}

				} else {

					if findHrefMarkEnd() {
						dropHeadUntil(hrefMarkEndPos)
						continue
					} else {
						l := len(buf) - (len(hrefStart) - 1)
						if l > 0 {
							dropHeadUntil(l)
						}
						break
					}

				}
			}

			if eof {
				return nil
			}
		}

	}
}
