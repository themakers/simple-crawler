package main

import (
	"context"
	"github.com/themakers/simple-crawler/crawler"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go signalHandler(ctx, cancel)

	go func() {
		defer cancel()

		var (
			crawled = map[string]bool{}
			crawledLock sync.Mutex
		)

		crawler.Crawl(ctx, []string{
			"https://themake.rs",
			"https://en.wikipedia.org/wiki/TCP_congestion_control",
		}, crawler.StreamingLinksFilter(1024), func(depth, pos int, origin string, link *url.URL) bool {

			linkText := link.String()
			link.Fragment = ""
			linkKey := link.String()


			crawledLock.Lock()
			defer crawledLock.Unlock()

			if link.Scheme != "data" {
				if crawled[linkKey] {
					return false
				} else {
					crawled[linkKey] = true
					log.Printf("url found. origin = %s, pos = %d, link = %s;", origin, pos, linkText)
					return true
				}
			} else {
				log.Printf("data url found. origin = %s, pos = %d;", origin, pos)
				return false
			}
		}, func(origin, link string, pos int, err error) {
			log.Printf("error. origin = %s, link = %s, pos = %d, error = %v;", origin, link, pos, err)
		}, crawler.Options{
			Depth: 2,
		})
	}()

	<-ctx.Done()
	log.Println("exiting:", ctx.Err())
	time.Sleep(1 * time.Second)
}

func signalHandler(ctx context.Context, cancel context.CancelFunc) {
	//> usually systemd sends SIGTERM, waits 90 secs, then sends SIGKILL

	c := make(chan os.Signal, 1)

	signal.Notify(c)

	for {
		select {
		case s := <-c:
			switch s {
			case os.Interrupt, syscall.SIGTERM, syscall.SIGHUP:
				log.Println("interrupt signal caught:", s.String())
				cancel()
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
