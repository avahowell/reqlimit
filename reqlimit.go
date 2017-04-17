package reqlimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// limiter is a http.Handler that rate-limits a http.Handler on a per-IP basis.
type limiter struct {
	mu sync.Mutex

	requests     map[string](chan struct{})
	limitTimeout time.Duration
	limit        uint64

	nextHandler http.Handler
}

// New creates a new Limiter, limiting requests to the supplied handler to
// `limit` requests over duration of the supplied `timeout`.
func New(h http.Handler, limit uint64, timeout time.Duration) *limiter {
	return &limiter{
		requests:     make(map[string]chan struct{}),
		limitTimeout: timeout,
		limit:        limit,

		nextHandler: h,
	}
}

// Listener's ServeHTTP implements the http.Handler interface and checks if the
// remote host has exceeded the request limit. If it has, it returns a
// http.Error with http.StatusTooManyRequests. Otherwise, the protected handler
// will be called.
func (l *limiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// grab the requests channel for this remote host, or create it if it doesn't
	// exist
	l.mu.Lock()
	requests, exists := l.requests[remoteIP]
	if !exists {
		requests = make(chan struct{}, l.limit)
		l.requests[remoteIP] = requests
	}
	l.mu.Unlock()

	// drain the request channel after the limit timeout
	go func() {
		time.Sleep(l.limitTimeout)
		<-requests
	}()

	// add to the request channel, throw an error if it is currently full.
	select {
	case requests <- struct{}{}:
	default:
		http.Error(w, "request limit exceeded", http.StatusTooManyRequests)
		return
	}

	l.nextHandler.ServeHTTP(w, r)
}
