package filters

import (
	"context"
	"github.com/themakers/simple-crawler/crawler"
	"io"
	"strings"
	"sync"
)

// TODO Implement general purpose FSM-scanner and use it here

// FIXME Strange behaviour on larger chunks
// FIXME Strange behaviour on smaller chunks
func StreamingFSMLinksFilter(chunkSize int) crawler.FilterFunc {
	// FIXME Pool optimization does not work!
	var pool sync.Pool
	pool.New = func() interface{} {
		return make([]byte, 0, 32*1024) //> Average page size on the internet, IMO
	}

	return func(ctx context.Context, r io.Reader, yieldTitle func(pos int, title string) error, yieldLink func(pos int, link string) error) error {
		if chunkSize < 1024 {
			chunkSize = 1024
		}

		const (
			hrefStart  = "href"
			titleStart = "<title>"
			titleEnd   = "</title>"
			bodyStart  = "<body"
		)

		var (
			buf       = pool.Get().([]byte)
			str       = ""
			readTotal = 0

			titleStartPos = -1
			titleEndPos   = -1
			titleFound    = false

			bodyStartPos = -1
			bodyFound    = false

			hrefMarkEndPos = -1
			hrefValuePos   = -1

			eof = false
		)

		obuf := buf
		defer pool.Put(obuf)

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

			titleStartPos -= delta
			titleEndPos -= delta
			bodyStartPos -= delta
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

		findTitleStart := func() bool {
			if i := strings.Index(str, titleStart); i >= 0 {
				titleStartPos = i + len(titleStart)
				return true
			} else {
				return false
			}
		}

		findTitleEnd := func() bool {
			if i := strings.Index(str, titleEnd); i >= 0 {
				titleEndPos = i
				return true
			} else {
				return false
			}
		}

		findBodyStart := func() bool {
			if i := strings.Index(str, bodyStart); i >= 0 {
				bodyStartPos = i
				return true
			} else {
				return false
			}
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
				if !titleFound && titleStartPos < 0 {

					if findTitleStart() {
						dropHeadUntil(titleStartPos)
					} else {
						break
					}

				} else if !titleFound && titleStartPos >= 0 && titleEndPos < 0 {

					if findTitleEnd() {
						title := str[:titleEndPos]

						if err := yieldTitle(readTotal-len(str)+titleStartPos, title); err != nil {
							return err
						}

						titleEndPos += len(titleEnd)

						titleFound = true
						dropHeadUntil(titleEndPos)
					} else {
						break
					}

				} else if !bodyFound {

					if findBodyStart() {
						bodyStartPos += len(bodyStart)
						bodyFound = true
						dropHeadUntil(bodyStartPos)
					} else {
						break
					}

				} else if hrefValuePos >= 0 {

					if end := findHrefValueEnd(); end >= 0 {
						if err := yieldLink(readTotal-len(str)+hrefValuePos, str[:end]); err != nil {
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
