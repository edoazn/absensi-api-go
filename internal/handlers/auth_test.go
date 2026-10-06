package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/server"
	"github.com/edoazn/absensi-go/internal/testsupport"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newTestRouter(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := testsupport.TestConfig(t)
	_, db := testsupport.SetupDB(t)
	return db, server.BuildRouter(server.Deps{Cfg: cfg, DB: db})
}

func doJSON(t *testing.T, r *gin.Engine, method, path, bearer string, body any) (int, api.Envelope) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var env api.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response %s %s: %v — body: %s", method, path, err, rec.Body.String())
	}
	return rec.Code, env
}

func decodeAuthData(t *testing.T, env api.Envelope) dto.AuthData {
	t.Helper()
	raw, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data back: %v", err)
	}
	var data dto.AuthData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("decode auth data: %v", err)
	}
	return data
}

func loginRequest(t *testing.T, r *gin.Engine, identity, password string) (int, dto.AuthData, api.Envelope) {
	t.Helper()
	status, env := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identity_number": identity,
		"password":        password,
	})
	return status, decodeAuthData(t, env), env
}

func TestLoginSuccessReturnsTokensAndProfile(t *testing.T) {
	db, r := newTestRouter(t)
	testsupport.CreateUser(t, db, "Budi Santoso", "211420108", models.RoleMahasiswa, testsupport.PtrString("budi@test.ac.id"))

	status, data, env := loginRequest(t, r, "211420108", testsupport.TestPassword)

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %+v", status, env)
	}
	if !env.Success {
		t.Fatalf("success must be true, got: %+v", env)
	}
	if data.AccessToken == "" || data.RefreshToken == "" {
		t.Fatal("access_token and refresh_token must be present")
	}
	if data.TokenType != "Bearer" || data.ExpiresIn <= 0 {
		t.Fatalf("token_type/expires_in wrong: %+v", data)
	}
	if data.User.IdentityNumber != "211420108" || data.User.Role != models.RoleMahasiswa {
		t.Fatalf("profile mismatch: %+v", data.User)
	}
}

func TestLoginUnknownIdentity(t *testing.T) {
	_, r := newTestRouter(t)

	status, _, env := loginRequest(t, r, "999999999", testsupport.TestPassword)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
	if env.Success {
		t.Fatal("success must be false")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	db, r := newTestRouter(t)
	testsupport.CreateUser(t, db, "Budi Santoso", "211420108", models.RoleMahasiswa, nil)

	status, _, _ := loginRequest(t, r, "211420108", "password-salah")

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestLoginValidationErrors(t *testing.T) {
	_, r := newTestRouter(t)

	status, env := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identity_number": "",
	})

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", status)
	}
	if env.Errors == nil {
		t.Fatal("errors bag must be present for validation failure")
	}
}

func TestMeRequiresBearer(t *testing.T) {
	_, r := newTestRouter(t)

	status, env := doJSON(t, r, http.MethodGet, "/api/v1/me", "", nil)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
	if env.Message != "Unauthenticated" {
		t.Fatalf("message = %q, want Unauthenticated", env.Message)
	}
}

func TestMeWithGarbageToken(t *testing.T) {
	_, r := newTestRouter(t)

	status, _ := doJSON(t, r, http.MethodGet, "/api/v1/me", "ini-bukan-token", nil)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestMeWithValidToken(t *testing.T) {
	db, r := newTestRouter(t)
	testsupport.CreateUser(t, db, "Siti Rahayu", "211420109", models.RoleMahasiswa, nil)

	status, _, env := loginRequest(t, r, "211420109", testsupport.TestPassword)
	if status != http.StatusOK {
		t.Fatalf("login failed: %d %+v", status, env)
	}

	meStatus, meEnv := doJSON(t, r, http.MethodGet, "/api/v1/me", decodeAuthData(t, env).AccessToken, nil)
	if meStatus != http.StatusOK {
		t.Fatalf("me status = %d, want 200; body %+v", meStatus, meEnv)
	}
	raw, _ := json.Marshal(meEnv.Data)
	var profile dto.UserProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if profile.Name != "Siti Rahayu" || profile.IdentityNumber != "211420109" {
		t.Fatalf("profile mismatch: %+v", profile)
	}
}

func TestRefreshRotatesAndInvalidatesOldToken(t *testing.T) {
	db, r := newTestRouter(t)
	testsupport.CreateUser(t, db, "Budi Santoso", "211420108", models.RoleMahasiswa, nil)

	status, first, env := loginRequest(t, r, "211420108", testsupport.TestPassword)
	if status != http.StatusOK {
		t.Fatalf("login failed: %d %+v", status, env)
	}

	refreshStatus, refreshedEnv := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
		"refresh_token": first.RefreshToken,
	})
	if refreshStatus != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200; body %+v", refreshStatus, refreshedEnv)
	}
	second := decodeAuthData(t, refreshedEnv)
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("rotated refresh token must differ from the old one")
	}

	reuseStatus, reuseEnv := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
		"refresh_token": first.RefreshToken,
	})
	if reuseStatus != http.StatusUnauthorized {
		t.Fatalf("reusing old refresh token must be 401, got %d; body %+v", reuseStatus, reuseEnv)
	}

	finalStatus, finalEnv := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
		"refresh_token": second.RefreshToken,
	})
	if finalStatus != http.StatusOK {
		t.Fatalf("newest refresh token must still work, got %d; body %+v", finalStatus, finalEnv)
	}
}

func TestLogoutBlacklistsAccessAndRevokesRefresh(t *testing.T) {
	db, r := newTestRouter(t)
	testsupport.CreateUser(t, db, "Budi Santoso", "211420108", models.RoleMahasiswa, nil)

	status, session, env := loginRequest(t, r, "211420108", testsupport.TestPassword)
	if status != http.StatusOK {
		t.Fatalf("login failed: %d %+v", status, env)
	}

	logoutStatus, logoutEnv := doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", session.AccessToken, map[string]string{
		"refresh_token": session.RefreshToken,
	})
	if logoutStatus != http.StatusOK {
		t.Fatalf("logout status = %d, want 200; body %+v", logoutStatus, logoutEnv)
	}

	if meStatus, _ := doJSON(t, r, http.MethodGet, "/api/v1/me", session.AccessToken, nil); meStatus != http.StatusUnauthorized {
		t.Fatalf("access token after logout must be blacklisted, got %d", meStatus)
	}

	refreshStatus, _ := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{
		"refresh_token": session.RefreshToken,
	})
	if refreshStatus != http.StatusUnauthorized {
		t.Fatalf("refresh token after logout must be revoked, got %d", refreshStatus)
	}
}

func TestLoginRateLimited(t *testing.T) {
	db, r := newTestRouter(t)
	testsupport.CreateUser(t, db, "Budi Santoso", "211420108", models.RoleMahasiswa, nil)

	last := 0
	for i := 0; i < 11; i++ {
		status, _, _ := loginRequest(t, r, "211420108", "password-salah")
		last = status
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("login request #11 = %d, want 429 (rate limited)", last)
	}
}
