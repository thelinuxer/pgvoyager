package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thelinuxer/pgvoyager/internal/selfupdate"
)

func TestUpdateStatusServerEdition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetUpdateManager(nil) // server edition
	r := gin.New()
	r.GET("/api/update/status", UpdateStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/update/status", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["edition"] != "server" {
		t.Fatalf("edition = %v, want server", body["edition"])
	}
}

func TestUpdateRestartRejectedWithoutManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetUpdateManager(nil)
	r := gin.New()
	r.POST("/api/update/restart", UpdateRestart)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/update/restart", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status code = %d, want 409", w.Code)
	}
}

func TestUpdateStatusDesktopEdition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := selfupdate.NewManager("1.0.0")
	SetUpdateManager(m)
	t.Cleanup(func() { SetUpdateManager(nil) })
	r := gin.New()
	r.GET("/api/update/status", UpdateStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/update/status", nil)
	r.ServeHTTP(w, req)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["edition"] != "desktop" {
		t.Fatalf("edition = %v, want desktop", body["edition"])
	}
}
