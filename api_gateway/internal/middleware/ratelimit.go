package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	requests int
	window   time.Duration
	mu       sync.Mutex
	clients  map[string]*clientWindow
}

type clientWindow struct {
	count   int
	expires time.Time
}

func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	if requests <= 0 || window <= 0 {
		return &RateLimiter{}
	}

	return &RateLimiter{
		requests: requests,
		window:   window,
		clients:  make(map[string]*clientWindow),
	}
}

func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	if r == nil || r.requests == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		key := clientKey(req)

		if exceeded := r.hit(key); exceeded {
			w.Header().Set("Retry-After", strconv.Itoa(int(r.window.Seconds())))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func (r *RateLimiter) hit(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	window := r.window

	state, ok := r.clients[key]
	if !ok || now.After(state.expires) {
		r.clients[key] = &clientWindow{
			count:   1,
			expires: now.Add(window),
		}
		return false
	}

	if state.count >= r.requests {
		return true
	}

	state.count++
	return false
}

func clientKey(r *http.Request) string {
	if r == nil {
		return "unknown"
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}
