package middleware

import (
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	count    int
	windowAt time.Time
}

// RateLimit returns a middleware allowing at most maxReq requests per window
// per remote IP address.
func RateLimit(maxReq int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)

	// Periodically clean up stale entries.
	ticker := time.NewTicker(window * 2)
	go func() {
		for range ticker.C {
			mu.Lock()
			cutoff := time.Now().Add(-window)
			for ip, v := range visitors {
				if v.windowAt.Before(cutoff) {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			mu.Lock()
			v, ok := visitors[ip]
			if !ok || time.Since(v.windowAt) > window {
				visitors[ip] = &visitor{count: 1, windowAt: time.Now()}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}
			v.count++
			if v.count > maxReq {
				mu.Unlock()
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
