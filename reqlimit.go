package reqlimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type Limiter struct {
	mu sync.Mutex

	requests       map[string][]time.Time
	limit          uint64
	limitTimeout   time.Duration
	limitErrorText string

	nextHandler http.Handler
}

func New(h http.Handler, limit uint64, timeout time.Duration) *Limiter {
	return &Limiter{
		requests:       make(map[string][]time.Time),
		limit:          limit,
		limitTimeout:   timeout,
		limitErrorText: "exceeded request limit",

		nextHandler: h,
	}
}

func (l *Limiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, l.limitErrorText, http.StatusForbidden)
		return
	}

	l.nextHandler.ServeHTTP(w, r)
}
