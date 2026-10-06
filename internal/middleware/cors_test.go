package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/gin-gonic/gin"
)

func newCorsRouter(allowed []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS(allowed))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestCorsWildcardEchoesOrigin(t *testing.T) {
	r := newCorsRouter([]string{"*"})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if acao := rec.Header().Get("Access-Control-Allow-Origin"); acao != "http://localhost:5173" {
		t.Fatalf("ACAO = %q, want echoed origin", acao)
	}
}

func TestCorsPreflightReturns204WithHeaders(t *testing.T) {
	r := newCorsRouter([]string{"http://localhost:5173", "https://admin.kampus.ac.id"})

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://admin.kampus.ac.id")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if acao := rec.Header().Get("Access-Control-Allow-Origin"); acao != "https://admin.kampus.ac.id" {
		t.Fatalf("ACAO = %q", acao)
	}
	if methods := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "POST") {
		t.Fatalf("Allow-Methods = %q", methods)
	}
}

func TestCorsDisallowedOriginGetsNoHeaders(t *testing.T) {
	r := newCorsRouter([]string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("request itself must still be served, got %d", rec.Code)
	}
	if acao := rec.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Fatalf("disallowed origin must not receive ACAO, got %q", acao)
	}
}

func TestCorsNoOriginPassesThrough(t *testing.T) {
	r := newCorsRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if acao := rec.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Fatalf("no origin must yield no ACAO, got %q", acao)
	}
}
