package security

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAllowedOrigin(t *testing.T) {
	cases := []struct {
		name        string
		origin      string
		requestHost string
		want        bool
	}{
		{"empty origin allowed", "", "localhost:5137", true},
		{"same host allowed", "http://localhost:5137", "localhost:5137", true},
		{"loopback IPv4 allowed", "http://127.0.0.1:5137", "localhost:5137", true},
		{"loopback IPv6 allowed", "http://[::1]:5137", "localhost:5137", true},
		{"dev origin allowed", "http://localhost:5173", "localhost:5137", true},
		{"public host rejected", "http://evil.example.com", "localhost:5137", false},
		{"public IP rejected", "http://198.51.100.42:5137", "localhost:5137", false},
		{"private LAN host rejected", "http://192.168.1.10:5137", "localhost:5137", false},
		{"malformed origin rejected", "not a url", "localhost:5137", false},
		{"missing scheme rejected", "evil.example.com", "localhost:5137", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AllowedOrigin(tc.origin, tc.requestHost)
			if got != tc.want {
				t.Errorf("AllowedOrigin(%q, %q) = %v, want %v", tc.origin, tc.requestHost, got, tc.want)
			}
		})
	}
}

func TestIsLoopback(t *testing.T) {
	cases := map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		"127.5.5.5": true,
		"::1":       true,
		"0.0.0.0":   false,
		"8.8.8.8":   false,
		"":          false,
	}
	for host, want := range cases {
		if got := IsLoopback(host); got != want {
			t.Errorf("IsLoopback(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestListenHostDefault(t *testing.T) {
	t.Setenv("PGVOYAGER_HOST", "")
	if got := ListenHost(); got != "127.0.0.1" {
		t.Errorf("ListenHost default = %q, want 127.0.0.1 (must not bind all interfaces)", got)
	}
}

func TestListenHostOverride(t *testing.T) {
	t.Setenv("PGVOYAGER_HOST", "0.0.0.0")
	if got := ListenHost(); got != "0.0.0.0" {
		t.Errorf("ListenHost with override = %q, want 0.0.0.0", got)
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	required := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "no-referrer",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for k, v := range required {
		if got := w.Header().Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("CSP missing default-src 'self': %q", csp)
	}
	// SvelteKit emits an inline bootstrap <script>; the CSP must allow it
	// or the SPA loads HTML but never hydrates (the regression that
	// surfaced the day desktop wrapper was first opened).
	if !strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Errorf("CSP must allow inline scripts for SvelteKit hydration: %q", csp)
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP must forbid framing: %q", csp)
	}
}

func TestMaxBodyBytesRejectsOversize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodyBytes(16))
	r.POST("/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.Data(http.StatusOK, "application/octet-stream", body)
	})

	big := strings.Repeat("x", 1024)
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(big))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversize body got status %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestMaxBodyBytesSkipsTerminalWebSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodyBytes(16))
	r.POST("/api/claude/terminal/:id", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Data(http.StatusOK, "application/octet-stream", body)
	})

	big := strings.Repeat("x", 1024)
	req := httptest.NewRequest(http.MethodPost, "/api/claude/terminal/abc", strings.NewReader(big))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("terminal route should bypass body cap, got status %d", w.Code)
	}
}

func TestOriginGuardRejectsCrossOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OriginGuard())
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Host = "localhost:5137"
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("cross-origin got status %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestOriginGuardAllowsSameOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OriginGuard())
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Host = "localhost:5137"
	req.Header.Set("Origin", "http://localhost:5137")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("same-origin got status %d, want %d", w.Code, http.StatusOK)
	}
}

// TestOriginGuardHostValidation verifies that OriginGuard rejects DNS-rebind
// attempts (non-loopback Host header) and allows loopback and empty Hosts.
func TestOriginGuardHostValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"loopback Host allowed", "127.0.0.1:5137", http.StatusOK},
		{"localhost Host allowed", "localhost:5137", http.StatusOK},
		{"empty Host allowed", "", http.StatusOK},
		{"DNS-rebind host rejected", "evil.example.com:5137", http.StatusForbidden},
		{"DNS-rebind host no port rejected", "pgvoyager.attacker.com", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure PGVOYAGER_HOST is default (127.0.0.1) so ListenHost()
			// doesn't accidentally match the attacker host.
			t.Setenv("PGVOYAGER_HOST", "127.0.0.1")

			r := gin.New()
			r.Use(OriginGuard())
			r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

			req := httptest.NewRequest(http.MethodGet, "/ok", nil)
			req.Host = tc.host
			// No Origin header — exercises Host-only path
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Host=%q: got status %d, want %d", tc.host, w.Code, tc.wantStatus)
			}
		})
	}
}

// TestMain neutralizes any inherited PGVOYAGER_HOST so ListenHostDefault
// reflects the package default.
func TestMain(m *testing.M) {
	_ = os.Unsetenv("PGVOYAGER_HOST")
	os.Exit(m.Run())
}
