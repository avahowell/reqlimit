package reqlimit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestLimitHandler(t *testing.T) {
	maxRequests := uint64(10)
	duration := time.Second * 5

	limiter := New(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "success")
	}), maxRequests, duration)

	ts := httptest.NewServer(limiter)
	defer ts.Close()

	for i := 0; uint64(i) < maxRequests; i++ {
		_, err := http.Get(ts.URL)
		if err != nil {
			t.Fatal(err)
		}
	}

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusForbidden {
		t.Fatal("limiter did not return forbidden after exceeding maxrequests")
	}

	time.Sleep(duration)

	res, err = http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatal("limiter did not return StatusOK after waiting for the duration to expire")
	}
}

func BenchmarkLimitedHandler(b *testing.B) {
	maxRequests := uint64(10000)
	duration := time.Second

	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	limitedHandler := New(testHandler, maxRequests, duration)

	for i := 0; i < b.N; i++ {
		limitedHandler.ServeHTTP(w, r)
	}
}

func BenchmarkHandler(b *testing.B) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	for i := 0; i < b.N; i++ {
		testHandler.ServeHTTP(w, r)
	}
}
