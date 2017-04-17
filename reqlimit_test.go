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

	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatal("limiter did not return TooManyRequests after exceeding maxrequests")
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

func testRequest(remoteAddr string) (*httptest.ResponseRecorder, *http.Request) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	return w, r
}

func TestRequestLimitHandlerDifferentIPs(t *testing.T) {
	maxRequests := uint64(10)
	duration := time.Minute

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	limitedHandler := New(testHandler, maxRequests, duration)

	// fill up the limit
	for i := 0; uint64(i) < maxRequests; i++ {
		limitedHandler.ServeHTTP(testRequest("3.4.5.6:7483"))
	}

	// verify that the limit is hit
	w, r := testRequest("3.4.5.6:8080")
	limitedHandler.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatal("expected the limit to be hit")
	}

	// verify that we can successfully request with a different ip
	w, r = testRequest("1.2.3.4:1234")
	limitedHandler.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("expected to get StatusOK after changing IP")
	}
}

func BenchmarkLimitedHandler(b *testing.B) {
	b.ReportAllocs()

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
	b.ReportAllocs()

	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	for i := 0; i < b.N; i++ {
		testHandler.ServeHTTP(w, r)
	}
}
