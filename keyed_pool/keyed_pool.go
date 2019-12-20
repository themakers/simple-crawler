package keyed_pool

import "sync"

// TODO make Waiter a base method to make code more well-organized

type EnqueueFunc func(key string, fn func())
type WaiterFunc func(key string) func()

func Pool(threadsPerKey int) EnqueueFunc {
	var (
		cond = sync.NewCond(new(sync.Mutex))

		keys = map[string]int{}
	)

	return func(key string, fn func()) {
		defer cond.Broadcast()

		defer func() {
			cond.L.Lock()
			defer cond.L.Unlock()

			keys[key]--
			if keys[key] == 0 {
				delete(keys, key)
			}
		}()

		cond.L.Lock()
		for keys[key] >= threadsPerKey {
			cond.Wait()
		}

		keys[key]++
		cond.L.Unlock()

		if fn != nil {
			fn()
		}
	}
}

func Waiter(threadsPerKey int) WaiterFunc {
	do := Pool(threadsPerKey)

	return func(key string) func() {
		var (
			c1 = make(chan struct{})
			c2 = make(chan struct{})
		)

		go do(key, func() {
			close(c1)
			<-c2
		})

		<-c1

		return func() {
			close(c2)
		}
	}
}
