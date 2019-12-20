package main

import (
	"context"
	"encoding/json"
	"github.com/themakers/simple-crawler/crawler"
	"github.com/themakers/simple-crawler/filters"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go signalHandler(ctx, cancel)

	var (
		wg           sync.WaitGroup
		crawled      = map[string]bool{}
		crawledHosts = map[string]bool{}
		crawledLock  sync.Mutex
		tt           int64
	)

	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()

		t0 := time.Now()

		cr := crawler.New(
			filters.StreamingFSMLinksFilter(1024),
			func(depth, pos int, origin string, title string) {
				log.Printf("title found. origin = %s, pos = %d, title = %s;", origin, pos, title)
			},
			func(depth, pos int, origin string, originalLink string, link *url.URL, external bool) bool {

				linkText := link.String()
				link.Fragment = ""
				linkKey := link.String()

				crawledLock.Lock()
				defer crawledLock.Unlock()

				crawledHosts[link.Host] = true

				if link.Scheme != "data" {
					if crawled[linkKey] {
						return false
					} else {
						crawled[linkKey] = true
						log.Printf("url found. external = %t; originalLink = %s; origin = %s, pos = %d, link = %s;", external, originalLink, origin, pos, linkText)

						atomic.StoreInt64(&tt, int64(time.Now().Sub(t0)))
						return true
					}
				} else {
					log.Printf("data url found. origin = %s, pos = %d;", origin, pos)
					return false
				}
			}, func(origin, link string, pos int, err error) {
				if !strings.Contains(err.Error(), context.Canceled.Error()) {
					log.Printf("error. origin = %s, link = %s, pos = %d, error = %v;", origin, link, pos, err)
				}
			},
			crawler.Options{
				Depth:                   0,
				ParallelRequestsPerHost: 0,
			},
		)

		cr.Feed(ctx, 3,
			"http://dummy_server:7532/",
		)

	}()

	<-ctx.Done()
	log.Println("exiting:", ctx.Err())

	wg.Wait()

	if data, err := json.MarshalIndent(crawledHosts, "", "  "); err != nil {
		panic(err)
	} else {
		log.Println(string(data))
		log.Println("Hosts total", len(crawledHosts))
		log.Println("Links total", len(crawled))
		log.Println("Time spent", time.Duration(tt))
		log.Println("Current speed", len(crawled)*int(time.Second)/int(tt), "links/sec")
	}
}

func signalHandler(ctx context.Context, cancel context.CancelFunc) {
	//> usually systemd sends SIGTERM, waits 90 secs, then sends SIGKILL

	c := make(chan os.Signal, 1)

	signal.Notify(c)

	sigintsCount := 0

	for {
		select {
		case s := <-c:
			switch s {
			case os.Interrupt, syscall.SIGTERM, syscall.SIGHUP:
				log.Println("interrupt signal caught:", s.String())
				cancel()
				sigintsCount++
				if sigintsCount >= 3 {
					log.Println("too much interrupts, exiting with error:", s.String())
					os.Exit(1)
				}
			case os.Kill:
				log.Println("kill signal caught")
				cancel() //> In case Kill was the first signal

				go func() {
					time.Sleep(2 * time.Second)
					os.Exit(1)
				}()
			default:
			}

			//> dont need this, going to handle more signals after SIGTERM/SIGINT
			//case <-ctx.Done():
			//	return
		}
	}
}
