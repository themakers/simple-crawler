package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// As we can see from benchmarks, regexp version is the slowest with
// precached page, and we can suppose that it will also have lowest
// performance when dealing with streams.

type FilterFunc func(ctx context.Context, r io.Reader, yield func(pos int, link string) error) error

type YieldURLFunc func(depth, pos int, origin string, link *url.URL) bool
type YieldErrorFunc func(origin, link string, pos int, err error)

type Options struct {
	UserAgent string

	// Zero for unlimited depth
	Depth int

	HTTPClientTimeout time.Duration
}

func Crawl(ctx context.Context, links []string, filter FilterFunc, yieldURL YieldURLFunc, yieldError YieldErrorFunc, ops Options) {
	if ops.UserAgent == "" {
		ops.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:73.0) Gecko/20100101 Firefox/73.0"
	}

	if ops.HTTPClientTimeout == 0 {
		ops.HTTPClientTimeout = 10 * time.Second
	}

	// TODO Make jar persistent
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		// TODO
		Jar:     jar,
		Timeout: ops.HTTPClientTimeout,
	}

	depth := ops.Depth

	var wg sync.WaitGroup
	wg.Add(len(links))
	for _, link := range links {
		go func(link string) {
			defer wg.Done()
			crawl(ctx, client, filter, depth, link, "", yieldURL, yieldError, ops)
		}(link)
	}
	wg.Wait()
}

func crawl(ctx context.Context, client *http.Client, filter FilterFunc, depth int, link, referer string, yieldURL YieldURLFunc, yieldError YieldErrorFunc, ops Options) {
	//log.Println("crawling link", link)
	req, err := newRequest(ctx, link, referer, ops)
	if err != nil {
		yieldError(link, "", -1, err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		yieldError(link, "", -1, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		yieldError(link, "", -1, errors.New(fmt.Sprintf("bad response status: %d", resp.StatusCode)))
		return
	}

	contentType := resp.Header.Get("Content-Type")

	//id := xid.New().String()

	if strings.HasPrefix(contentType, "text/html") {
		log.Println("ct:", link, contentType)
		var wg sync.WaitGroup
		defer wg.Wait()

		if err := filter(ctx, resp.Body, func(pos int, crawledURL string) error {

			u, err := url.Parse(link)
			if err != nil {
				yieldError(link, crawledURL, -1, err)
				return err
			}

			u, err = u.Parse(crawledURL)
			if err != nil {
				yieldError(link, crawledURL, pos, err)
				return err
			}

			doWeNeedThisLink := yieldURL(depth, pos, link, &(*u))

			u.Fragment = ""
			crawledURL = u.String()

			if doWeNeedThisLink {

				if (u.Scheme == "" || u.Scheme == "http" || u.Scheme == "https") && (ops.Depth == 0 || depth > 1) {
					wg.Add(1)
					go func() {
						defer wg.Done()

						crawl(ctx, client, filter, depth-1, crawledURL, link, yieldURL, yieldError, ops)
					}()
				}

			}

			return nil
		}); err != nil {
			return
		}

	} else if contentType == "" {
		// TODO Try to detect CT automatically
		//http.DetectContentType()
		log.Println("empty ct:", link)
	} else {
		//> Don't need to parse
		log.Println("bad ct:", link, contentType)
	}
}

func newRequest(ctx context.Context, link, referer string, ops Options) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	// TODO
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	//> Let's try to mimic a regular web browser
	req.Header.Set("User-Agent", ops.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", u.Host)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	return req, err
}
