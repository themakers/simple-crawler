package crawler

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
)

// Run with: go test -v -benchmem -bench=. .

var data func() io.Reader

func TestMain(m *testing.M) {
	resp, err := http.Get("https://en.wikipedia.org/wiki/NOP_(code)")
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	data = func() io.Reader {
		return bytes.NewReader(body)
	}

	log.Println("Page Size", len(body))

	m.Run()
}

func BenchmarkStreamingRegexpLinksFilter(b *testing.B) {

	for i := 0; i < b.N; i++ {
		if err := RegexpLinksFilter()(context.Background(), data(), func(pos int, link string) error {
			return nil
		}); err != nil {
			panic(err)
		}
	}

}

func BenchmarkStreamingLinksFilter1024(b *testing.B) {

	for i := 0; i < b.N; i++ {
		if err := StreamingLinksFilter(1024)(context.Background(), data(), func(pos int, link string) error {
			return nil
		}); err != nil {
			panic(err)
		}
	}

}

func BenchmarkStreamingGoHTMLLinksFilter(b *testing.B) {

	for i := 0; i < b.N; i++ {
		if err := StreamingGoHTMLLinksFilter()(context.Background(), data(), func(pos int, link string) error {
			return nil
		}); err != nil {
			panic(err)
		}
	}

}

func BenchmarkGoQueryLinksFilter(b *testing.B) {

	for i := 0; i < b.N; i++ {
		if err := GoQueryLinksFilter()(context.Background(), data(), func(pos int, link string) error {
			return nil
		}); err != nil {
			panic(err)
		}
	}

}

func TestStreamingRegexpLinksFilter(t *testing.T) {
	nlinks := 0
	if err := RegexpLinksFilter()(context.Background(), data(), func(pos int, link string) error {
		nlinks += 1
		t.Log("RegexpLinksFilter => ", link)
		return nil
	}); err != nil {
		panic(err)
	}
	t.Log("RegexpLinksFilter == ", nlinks)
}

func TestStreamingLinksFilter(t *testing.T) {
	nlinks := 0
	if err := StreamingLinksFilter(1024)(context.Background(), data(), func(pos int, link string) error {
		nlinks += 1
		t.Log("StreamingLinksFilter => ", link)
		return nil
	}); err != nil {
		panic(err)
	}
	t.Log("StreamingLinksFilter == ", nlinks)
}

func TestStreamingGoHTMLLinksFilter(t *testing.T) {
	nlinks := 0
	if err := StreamingGoHTMLLinksFilter()(context.Background(), data(), func(pos int, link string) error {
		nlinks += 1
		t.Log("StreamingGoHTMLLinksFilter => ", link)
		return nil
	}); err != nil {
		panic(err)
	}
	t.Log("StreamingGoHTMLLinksFilter == ", nlinks)
}

func TestGoQueryLinksFilter(t *testing.T) {
	nlinks := 0
	if err := GoQueryLinksFilter()(context.Background(), data(), func(pos int, link string) error {
		nlinks += 1
		t.Log("GoQueryLinksFilter => ", link)
		return nil
	}); err != nil {
		panic(err)
	}
	t.Log("GoQueryLinksFilter == ", nlinks)
}
