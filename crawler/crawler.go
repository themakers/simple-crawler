package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

type FilterFunc func(ctx context.Context, r io.Reader, yieldTitle func(pos int, title string) error, yieldLink func(pos int, link string) error) error

type YieldURLFunc func(depth, pos int, origin string, originalLink string, link *url.URL, external bool) bool
type YieldTitleFunc func(depth, pos int, origin string, title string)
type YieldErrorFunc func(origin, link string, pos int, err error)

type Options struct {
	Client *http.Client

	UserAgent string

	// Zero for unlimited depth
	Depth int

	// Zero for no limit
	ParallelRequestsPerHost int
}

type Crawler struct {
	filter     FilterFunc
	yieldURL   YieldURLFunc
	yieldTitle YieldTitleFunc
	yieldError YieldErrorFunc
	ops        Options

	request func(q *http.Request) (*http.Response, error)
}

func New(filter FilterFunc, yieldTitle YieldTitleFunc, yieldURL YieldURLFunc, yieldError YieldErrorFunc, ops Options) *Crawler {
	cr := &Crawler{
		filter:     filter,
		yieldTitle: yieldTitle,
		yieldURL:   yieldURL,
		yieldError: yieldError,
		ops:        ops,
	}

	if cr.ops.UserAgent == "" {
		cr.ops.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:73.0) Gecko/20100101 Firefox/73.0"
	}

	if cr.ops.Client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			panic(err) //> No need to handle this error properly, 'cause it always nil
		}

		cr.ops.Client = &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		}
	}

	if cr.ops.ParallelRequestsPerHost == 0 {
		cr.request = func(q *http.Request) (resp *http.Response, err error) {
			return cr.ops.Client.Do(q)
		}
	} else {
		cr.request = newRequestPool(cr.ops.Client, cr.ops.ParallelRequestsPerHost)
	}

	return cr
}

func (cr *Crawler) Feed(ctx context.Context, depth int, links ...string) {
	if depth == 0 {
		depth = cr.ops.Depth
	}

	var wg sync.WaitGroup
	wg.Add(len(links))
	for _, link := range links {
		go func(link string) {
			defer wg.Done()
			cr.handle(ctx, depth, depth, link, "")
		}(link)
	}
	wg.Wait()
}

func (cr *Crawler) handle(ctx context.Context, initialDepth, depth int, link, referer string) {
	req, err := cr.newRequest(ctx, link, referer)
	if err != nil {
		cr.yieldError(link, "", -1, err)
		return
	}

	resp, err := cr.request(req)
	if err != nil {
		cr.yieldError(link, "", -1, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cr.yieldError(link, "", -1, errors.New(fmt.Sprintf("bad response status: %d", resp.StatusCode)))
		return
	}

	contentType := resp.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "text/html") {
		var wg sync.WaitGroup
		defer wg.Wait()

		if err := cr.filter(ctx, resp.Body, func(pos int, title string) error {

			cr.yieldTitle(depth, pos, link, title)

			return nil

		}, func(pos int, crawledLink string) error {

			originURL, err := url.Parse(link)
			if err != nil {
				cr.yieldError(link, crawledLink, -1, err)
				return err
			}

			crawledURL, err := url.Parse(crawledLink)
			if err != nil {
				cr.yieldError(link, crawledLink, pos, err)
				return err
			}


			absURL := originURL.ResolveReference(crawledURL)

			external := crawledURL.Host != "" && crawledURL.Host != originURL.Host

			doWeNeedThisLink := cr.yieldURL(depth, pos, link, crawledLink, &(*absURL), external)

			absURL.Fragment = ""

			if doWeNeedThisLink {

				if (absURL.Scheme == "" || absURL.Scheme == "http" || absURL.Scheme == "https") &&
					((initialDepth == 0 && cr.ops.Depth == 0) || (initialDepth != 0 && depth > 1)) {
					wg.Add(1)
					go func() {
						defer wg.Done()

						cr.handle(ctx, initialDepth, depth-1, absURL.String(), link)
					}()
				}

			}

			return nil
		}); err != nil {
			return
		}

	} else if contentType == "" {
		// TODO Try to detect CT automatically
		// http.DetectContentType()
	} else {
		//> Don't need to parse
	}
}

func (cr *Crawler) newRequest(ctx context.Context, link, referer string) (*http.Request, error) {
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
	req.Header.Set("User-Agent", cr.ops.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", u.Host)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	return req, err
}
