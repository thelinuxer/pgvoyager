package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadVerifiesAndStages(t *testing.T) {
	asset, err := AssetName()
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("new-binary-bytes")
	sum := sha256.Sum256(body)
	sumsLine := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), asset)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v9.9.9/"+asset:
			_, _ = w.Write(body)
		case r.URL.Path == "/v9.9.9/SHA256SUMS":
			_, _ = w.Write([]byte(sumsLine))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	origBase, origDir, origHost := baseURL, stagingDir, hostAllowed
	t.Cleanup(func() { baseURL, stagingDir, hostAllowed = origBase, origDir, origHost })
	baseURL = srv.URL + "/"
	hostAllowed = func(string) bool { return true } // httptest runs on 127.0.0.1
	dir := t.TempDir()
	stagingDir = func() (string, error) { return dir, nil }

	staged, err := Download(context.Background(), "v9.9.9", nil)
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	got, _ := os.ReadFile(staged)
	if string(got) != string(body) {
		t.Fatalf("staged content = %q, want %q", got, body)
	}
	if filepath.Dir(staged) != dir {
		t.Fatalf("staged in %s, want %s", filepath.Dir(staged), dir)
	}
}

func TestDownloadRejectsBadChecksum(t *testing.T) {
	asset, _ := AssetName()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v9.9.9/"+asset:
			_, _ = w.Write([]byte("real-bytes"))
		case r.URL.Path == "/v9.9.9/SHA256SUMS":
			_, _ = w.Write([]byte("deadbeef  " + asset + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	origBase, origDir, origHost := baseURL, stagingDir, hostAllowed
	t.Cleanup(func() { baseURL, stagingDir, hostAllowed = origBase, origDir, origHost })
	baseURL = srv.URL + "/"
	hostAllowed = func(string) bool { return true } // httptest runs on 127.0.0.1
	dir := t.TempDir()
	stagingDir = func() (string, error) { return dir, nil }

	if _, err := Download(context.Background(), "v9.9.9", nil); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("staged temp not cleaned up: %v", entries)
	}
}

func TestDefaultHostAllowed(t *testing.T) {
	allowed := []string{"github.com", "api.github.com", "codeload.github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com"}
	for _, h := range allowed {
		if !defaultHostAllowed(h) {
			t.Errorf("defaultHostAllowed(%q) = false, want true", h)
		}
	}
	denied := []string{"evil.com", "github.com.evil.com", "notgithub.com", "githubusercontent.com.evil.com", "127.0.0.1"}
	for _, h := range denied {
		if defaultHostAllowed(h) {
			t.Errorf("defaultHostAllowed(%q) = true, want false", h)
		}
	}
}

func TestDownloadRejectsDisallowedHost(t *testing.T) {
	origBase := baseURL
	t.Cleanup(func() { baseURL = origBase })
	// Real default allowlist is active here (not overridden); a non-GitHub
	// base URL must be refused before any network call.
	baseURL = "https://evil.example.com/"
	if _, err := Download(context.Background(), "v9.9.9", nil); err == nil {
		t.Fatalf("expected refusal for disallowed host")
	}
}
