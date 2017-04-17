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

	requests     map[string][]time.Time
	limit        uint64
	limitTimeout time.Duration

	nextHandler http.Handler
}

// New creates a new Limiter, limiting requests to the supplied handler to
// `limit` requests over duration of the supplied `timeout`.
func New(h http.Handler, limit uint64, timeout time.Duration) *limiter {
	return &limiter{
		requests:     make(map[string][]time.Time),
		limit:        limit,
		limitTimeout: timeout,

		nextHandler: h,
	}
}

// Listener's ServeHTTP implements the http.Handler interface and checks if the
// remote host has exceeded the request limit. If it has, it returns a
// http.Error with http.StatusForbidden. Otherwise, the protected handler will
// be called.
func (l *limiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()

	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var updatedHistory []time.Time
	history, exists := l.requests[remoteIP]

	if exists {
		// filter requests that have expired
		for _, requestTime := range history {
			if requestTime.Add(l.limitTimeout).After(time.Now()) {
				updatedHistory = append(updatedHistory, requestTime)
			}
		}
	}

	updatedHistory = append(updatedHistory, time.Now())
	l.requests[remoteIP] = updatedHistory

	if uint64(len(updatedHistory)) > l.limit {
		http.Error(w, "request limit exceeded", http.StatusTooManyRequests)
		return
	}

	l.nextHandler.ServeHTTP(w, r)
}
