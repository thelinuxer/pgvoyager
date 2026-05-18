package static

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func newServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fs := fstest.MapFS{
		"dist/index.html":     {Data: []byte("<html>spa</html>")},
		"dist/app.js":         {Data: []byte("console.log(1)")},
		"dist/secret/key.txt": {Data: []byte("topsecret")},
	}
	r := gin.New()
	r.Use(ServeEmbedded(fs, "dist"))
	r.GET("/api/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestServeIndexAtRoot(t *testing.T) {
	r := newServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "<html>spa</html>" {
		t.Errorf("/ got %d %q, want 200 <html>spa</html>", w.Code, w.Body.String())
	}
}

func TestServeAsset(t *testing.T) {
	r := newServer(t)
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "console.log(1)" {
		t.Errorf("/app.js got %d %q", w.Code, w.Body.String())
	}
}

func TestPassThroughAPI(t *testing.T) {
	r := newServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "pong" {
		t.Errorf("/api/ping got %d %q", w.Code, w.Body.String())
	}
}

func TestRejectPathTraversal(t *testing.T) {
	r := newServer(t)
	cases := []string{
		"/../etc/passwd",
		"/secret/../../../etc/passwd",
		"/%2e%2e/passwd",
	}
	for _, p := range cases {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusOK && w.Body.Len() > 0 && w.Body.String() != "<html>spa</html>" {
			t.Errorf("path %q returned non-SPA content with status %d: %q", p, w.Code, w.Body.String())
		}
	}
}
