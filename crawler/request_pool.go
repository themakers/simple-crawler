package crawler

import (
	"github.com/themakers/simple-crawler/keyed_pool"
	"net/http"
)

func newRequestPool(client *http.Client, requestsPerKey int) func(q *http.Request) (resp *http.Response, err error) {
	wait := keyed_pool.Waiter(requestsPerKey)

	return func(q *http.Request) (resp *http.Response, err error) {
		key := q.Host
		if key == "" {
			key = q.URL.Host
		}

		defer wait(key)()
		return client.Do(q)
	}
}
