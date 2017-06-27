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

	requests     map[string]uint64
	limitTimeout time.Duration
	limit        uint64

	nextHandler http.Handler
}

// New creates a new Limiter, limiting each remote host served by the supplied
// handler to `limit` requests over duration of the supplied `timeout`.
func New(h http.Handler, limit uint64, timeout time.Duration) *limiter {
	return &limiter{
		requests:     make(map[string]uint64),
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

	// lookup the number of active requests for this IP
	l.mu.Lock()
	limitExceeded := l.requests[remoteIP] >= l.limit
	if !limitExceeded {
		// increment the number of requests
		l.requests[remoteIP]++

		// after timeout, decrement the number of requests
		time.AfterFunc(l.limitTimeout, func() {
			l.mu.Lock()
			l.requests[remoteIP]--
			if l.requests[remoteIP] == 0 {
				delete(l.requests, remoteIP)
			}
			l.mu.Unlock()
		})
	}
	l.mu.Unlock()

	if limitExceeded {
		http.Error(w, "request limit exceeded", http.StatusTooManyRequests)
		return
	}

	l.nextHandler.ServeHTTP(w, r)
}
