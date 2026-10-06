package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/server"
	"github.com/edoazn/absensi-go/internal/services"
	"github.com/edoazn/absensi-go/internal/testsupport"
	"github.com/gin-gonic/gin"
)

func adminGuardRouter(t *testing.T) (*gin.Engine, *services.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := testsupport.TestConfig(t)
	_, db := testsupport.SetupDB(t)

	tokens := services.NewTokenService(cfg, db)
	blacklist := services.NewBlacklist()

	r := gin.New()
	r.GET("/guarded", middleware.JWTAuth(tokens, blacklist), middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r, tokens
}

func tokenFor(t *testing.T, tokens *services.TokenService, role string) string {
	t.Helper()
	user := &models.User{ID: 1, Name: "X", Role: role}
	raw, _, err := tokens.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return raw
}

func TestRequireAdminAllowsAdmin(t *testing.T) {
	r, tokens := adminGuardRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, tokens, models.RoleAdmin))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("admin must pass, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireAdminRejectsMahasiswa(t *testing.T) {
	r, tokens := adminGuardRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, tokens, models.RoleMahasiswa))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("mahasiswa must get 403, got %d", rec.Code)
	}
}

func TestJWTAuthRejectsMissingAndTampered(t *testing.T) {
	r, _ := adminGuardRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing bearer must be 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/guarded", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, services.NewTokenService(testsupport.TestConfig(t), nil), models.RoleAdmin)+"x")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tampered token must be 401, got %d", rec.Code)
	}
}

func TestBlacklistedTokenRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testsupport.TestConfig(t)
	_, db := testsupport.SetupDB(t)

	tokens := services.NewTokenService(cfg, db)
	blacklist := services.NewBlacklist()

	r := gin.New()
	r.GET("/guarded", middleware.JWTAuth(tokens, blacklist), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	raw := tokenFor(t, tokens, models.RoleAdmin)
	claims, err := tokens.ParseAccessToken(raw)
	if err != nil {
		t.Fatalf("parse own token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pre-blacklist request must pass, got %d", rec.Code)
	}

	blacklist.Add(claims.ID, claims.ExpiresAt.Time)

	req = httptest.NewRequest(http.MethodGet, "/guarded", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("blacklisted token must be 401, got %d", rec.Code)
	}
}

var _ = server.BuildRouter
