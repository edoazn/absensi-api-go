package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edoazn/absensi-go/config"
	"github.com/gin-gonic/gin"
)

func spaTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		AppName:    "spa-test",
		JWTSecret:  "test-secret-0123456789abcdef0123456789abcdef",
		BcryptCost: 4,
		Timezone:   time.UTC,
	}
	return BuildRouter(Deps{Cfg: cfg, DB: nil})
}

func writeSpaFixture(t *testing.T) string {
	t.Helper()
	// Handler membaca ./web/dist relatif ke working directory, jadi fixture
	// dibuat di bawah <tmp>/web/dist dan test berjalan dari <tmp>.
	root := filepath.Join(t.TempDir(), "web", "dist")
	write := func(rel, body string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "<html><body>SPA-INDEX</body></html>")
	write("favicon.svg", "<svg></svg>")
	write("assets/app-a1b2c3.js", "console.log('app')")
	return filepath.Dir(filepath.Dir(root)) // <tmp>
}

func serveRequest(router *gin.Engine, method, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, target, nil))
	return w
}

func TestSpaAdminWithFixture(t *testing.T) {
	restore := chdir(t, writeSpaFixture(t))
	defer restore()

	router := spaTestRouter(t)

	t.Run("root redirects to /admin/", func(t *testing.T) {
		w := serveRequest(router, http.MethodGet, "/")
		if w.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != "/admin/" {
			t.Fatalf("Location = %q, want /admin/", loc)
		}
	})

	t.Run("admin root serves index.html", func(t *testing.T) {
		w := serveRequest(router, http.MethodGet, "/admin")
		assertIndexServed(t, w, "/admin")
	})

	t.Run("deep link falls back to index.html", func(t *testing.T) {
		for _, path := range []string{"/admin/login", "/admin/users"} {
			w := serveRequest(router, http.MethodGet, path)
			assertIndexServed(t, w, path)
		}
	})

	t.Run("hashed asset served immutable", func(t *testing.T) {
		w := serveRequest(router, http.MethodGet, "/admin/assets/app-a1b2c3.js")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if !strings.Contains(w.Body.String(), "console.log") {
			t.Fatalf("body tidak berisi konten aset: %q", w.Body.String())
		}
		if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
			t.Fatalf("Cache-Control = %q, want immutable", cc)
		}
	})

	t.Run("non-asset file served no-cache", func(t *testing.T) {
		w := serveRequest(router, http.MethodGet, "/admin/favicon.svg")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Fatalf("Cache-Control = %q, want no-cache", cc)
		}
	})

	t.Run("missing asset with extension is plain 404", func(t *testing.T) {
		w := serveRequest(router, http.MethodGet, "/admin/missing.css")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("path outside admin is plain 404", func(t *testing.T) {
		for _, path := range []string{"/users", "/login", "/whatever"} {
			w := serveRequest(router, http.MethodGet, path)
			if w.Code != http.StatusNotFound {
				t.Fatalf("%s: status = %d, want 404", path, w.Code)
			}
		}
	})

	t.Run("unknown api returns json 404 envelope", func(t *testing.T) {
		w := serveRequest(router, http.MethodGet, "/api/v1/nonexistent")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		var env struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("body bukan JSON envelope: %v (%q)", err, w.Body.String())
		}
		if env.Success || env.Message == "" {
			t.Fatalf("envelope tak sesuai: success=%v message=%q", env.Success, env.Message)
		}
	})

	t.Run("path traversal is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
		req.URL.Path = "/admin/../../go.mod"
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		if strings.Contains(w.Body.String(), "module") {
			t.Fatalf("isi go.mod bocor: %q", w.Body.String())
		}
	})
}

func TestStaticAdminLegacyRedirects(t *testing.T) {
	restore := chdir(t, writeSpaFixture(t))
	defer restore()

	router := spaTestRouter(t)

	cases := []struct {
		method string
		in     string
		want   string
	}{
		{http.MethodGet, "/static/admin", "/admin/"},
		{http.MethodGet, "/static/admin/users", "/admin/users"},
		{http.MethodPost, "/static/admin/users", "/admin/users"},
		{http.MethodGet, "/static/admin/login", "/admin/login"},
		{http.MethodGet, "/static/admin/schedules/3/edit", "/admin/schedules"},
		{http.MethodGet, "/static/admin/unknown/x", "/admin/"},
	}
	for _, tc := range cases {
		w := serveRequest(router, tc.method, tc.in)
		if w.Code != http.StatusFound {
			t.Errorf("%s %s: status = %d, want 302", tc.method, tc.in, w.Code)
			continue
		}
		if loc := w.Header().Get("Location"); loc != tc.want {
			t.Errorf("%s %s: Location = %q, want %q", tc.method, tc.in, loc, tc.want)
		}
	}
}

func assertIndexServed(t *testing.T, w *httptest.ResponseRecorder, label string) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("%s: status = %d, want 200", label, w.Code)
	}
	if !strings.Contains(w.Body.String(), "SPA-INDEX") {
		t.Fatalf("%s: index.html tidak disajikan, body: %q", label, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("%s: Content-Type = %q, want text/html", label, ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("%s: Cache-Control = %q, want no-cache", label, cc)
	}
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatal(err)
		}
	}
}
