package keyed_pool

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestKeyedPool(t *testing.T) {
	const (
		delta = 50 * time.Millisecond
		delay = delta / 10
	)

	var (
		order     []string
		orderLock sync.Mutex

		wg sync.WaitGroup
	)

	putDone := func(n string) {
		defer wg.Done()

		orderLock.Lock()
		defer orderLock.Unlock()

		order = append(order, n)
	}

	t0 := time.Now()

	wg.Add(6)

	do := Pool(2)

	go do("1", func() {
		time.Sleep(1 * delta)
		putDone("11")
	})
	time.Sleep(delay)
	go do("1", func() {
		time.Sleep(6 * delta)
		putDone("12")
	})
	time.Sleep(delay)
	go do("1", func() {
		time.Sleep(3 * delta)
		putDone("13")
	})

	time.Sleep(delay)
	go do("2", func() {
		time.Sleep(3 * delta)
		putDone("21")
	})
	time.Sleep(delay)
	go do("2", func() {
		time.Sleep(2 * delta)
		putDone("22")
	})
	time.Sleep(delay)
	go do("2", func() {
		time.Sleep(3 * delta)
		putDone("23")
	})

	wg.Wait()

	tt := time.Now().Sub(t0)

	expected := []string{"11", "22", "21", "13", "23", "12"}
	if !reflect.DeepEqual(order, expected) {
		t.Log("bad execution order; actual", order, "expected", expected)
		t.Fail()
	}

	if !(tt >= delta*6 && tt <= delta*6+delta/2) {
		t.Log("bad test duration; actual", tt, "expected", delta*6)
		t.Fail()
	} else {
		t.Log("test duration", tt)
	}
}

func BenchmarkPool(b *testing.B) {

	var (
		wg sync.WaitGroup
		do = Pool(2)
	)

	for n := 0; n < b.N; n++ {
		wg.Add(1)
		go do("2", func() {
			wg.Done()
		})
	}

	wg.Wait()
}

func BenchmarkGoroutineAndWaitGroupOverhead(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			wg.Done()
		}()
		wg.Wait()
	}
}

func BenchmarkGoroutineAndChannelOverhead(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var ch = make(chan struct{})
		go func() {
			close(ch)
		}()
		<-ch
	}
}

func BenchmarkGoroutineAndContextOverhead(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			cancel()
		}()
		<-ctx.Done()
	}
}
