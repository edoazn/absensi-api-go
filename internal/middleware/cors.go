package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS mengizinkan permintaan lintas origin dari daftar origin yang
// dikonfigurasi (env CORS_ORIGINS, dipisah koma; "*" = semua).
// Mobile native tidak terpengaruh CORS — ini untuk browser dev (SPA/WebView).
// Origin di luar daftar tidak mendapat header CORS sama sekali.
func CORS(allowed []string) gin.HandlerFunc {
	wildcard := false
	set := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		if o == "*" {
			wildcard = true
			continue
		}
		set[strings.ToLower(strings.TrimSuffix(o, "/"))] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (wildcard || set[strings.ToLower(origin)]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
