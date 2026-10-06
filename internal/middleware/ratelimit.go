package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimit struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	interval time.Duration
	burst    int
}

func NewRateLimit(interval time.Duration, burst int) *RateLimit {
	return &RateLimit{
		visitors: make(map[string]*visitor),
		interval: interval,
		burst:    burst,
	}
}

func (rl *RateLimit) Middleware() gin.HandlerFunc {
	return rl.MiddlewareForKey(func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// MiddlewareForKey memakai token bucket per-key alih-alih per-IP —
// dipakai untuk rate limit per-USER pada rute yang sudah terautentikasi
// (satu IP bisa dipakai banyak user, dan satu user bisa ganti-ganti IP).
func (rl *RateLimit) MiddlewareForKey(keyFn func(c *gin.Context) string) gin.HandlerFunc {
	go rl.gcLoop()
	return func(c *gin.Context) {
		v := rl.getVisitor(keyFn(c))
		if !v.limiter.Allow() {
			api.Fail(c, http.StatusTooManyRequests, "Terlalu banyak permintaan, coba lagi nanti")
			return
		}
		c.Next()
	}
}

func (rl *RateLimit) getVisitor(key string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rate.Every(rl.interval), rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = time.Now()
	return v
}

func (rl *RateLimit) gcLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-3 * time.Minute)
		rl.mu.Lock()
		for key, v := range rl.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}
