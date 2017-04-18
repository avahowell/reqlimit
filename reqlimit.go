package reqlimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// limiter is a http.Handler that rate-limits a http.Handler on a per-IP basis.
type limiter struct {
	mu       sync.RWMutex // locks for requests
	requests map[string](chan struct{})

	limitTimeout time.Duration
	limit        uint64

	nextHandler http.Handler
}

// New creates a new Limiter, limiting each remote host served by the supplied
// handler to `limit` requests over duration of the supplied `timeout`.
func New(h http.Handler, limit uint64, timeout time.Duration) *limiter {
	return &limiter{
		requests:     make(map[string]chan struct{}),
		limitTimeout: timeout,
		limit:        limit,

		nextHandler: h,
	}
}

// getIP tries to find the real ip address of a client.
func getIP(r *http.Request) (string, error) {
	header := r.Header.Get("X-Real-Ip")
	realIP := strings.TrimSpace(header)
	if realIP != "" {
		return realIP, nil
	}

	realIP = r.Header.Get("X-Forwarded-For")
	idx := strings.IndexByte(realIP, ',')
	if idx >= 0 {
		realIP = realIP[0:idx]
	}
	realIP = strings.TrimSpace(realIP)
	if realIP != "" {
		return realIP, nil
	}

	addr := strings.TrimSpace(r.RemoteAddr)

	// if addr has port use the net.SplitHostPort otherwise(error occurs) take as it is
	ip, _, err := net.SplitHostPort(addr)
	return ip, err
}

// Listener's ServeHTTP implements the http.Handler interface and checks if the
// remote host has exceeded the request limit. If it has, it returns a
// http.Error with http.StatusTooManyRequests. Otherwise, the protected handler
// will be called.
func (l *limiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteIP, err := getIP(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// grab the requests channel for this remote host
	l.mu.RLock() // read lock
	requests, exists := l.requests[remoteIP]
	l.mu.RUnlock()

	// or create it if it doesn't exist
	if !exists {
		requests = make(chan struct{}, l.limit)

		l.mu.Lock() // write & read lock
		l.requests[remoteIP] = requests
		l.mu.Unlock()
	}

	// add to the request channel, throw an error if it is currently full.
	select {
	case requests <- struct{}{}:
		// drain the request channel after the limit timeout
		go func() {
			time.Sleep(l.limitTimeout)
			<-requests
		}()

	default:
		http.Error(w, "request limit exceeded", http.StatusTooManyRequests)
		return
	}

	l.nextHandler.ServeHTTP(w, r)
}
