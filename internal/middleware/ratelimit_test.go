package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestRateLimitBlocksBeyondBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := middleware.NewRateLimit(time.Minute, 3)
	r := gin.New()
	r.POST("/limited", rl.Middleware(), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	codes := make([]int, 5)
	for i := range codes {
		req := httptest.NewRequest(http.MethodPost, "/limited", nil)
		req.RemoteAddr = "192.0.2.10:5555"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		codes[i] = rec.Code
	}

	for i := range 3 {
		if codes[i] != http.StatusOK {
			t.Fatalf("request %d must pass within burst, got %d", i+1, codes[i])
		}
	}
	if codes[3] != http.StatusTooManyRequests || codes[4] != http.StatusTooManyRequests {
		t.Fatalf("requests beyond burst must be 429, got %v", codes[3:])
	}
}

func TestRateLimitIsolatedPerClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := middleware.NewRateLimit(time.Minute, 1)
	r := gin.New()
	r.POST("/limited", rl.Middleware(), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	for _, ip := range []string{"192.0.2.20:1111", "192.0.2.21:2222"} {
		req := httptest.NewRequest(http.MethodPost, "/limited", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("different IP %s must have its own bucket, got %d", ip, rec.Code)
		}
	}
}
