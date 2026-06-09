package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/thelinuxer/pgvoyager/internal/version"
)

// Overridable for tests.
var (
	baseURL    = "https://github.com/" + version.GitHubRepo + "/releases/download/"
	httpClient = &http.Client{Timeout: 5 * time.Minute}
	// stagingDir returns the directory to stage the download into — the
	// running executable's directory, so the later rename is on one filesystem.
	stagingDir = func() (string, error) {
		exe, err := exePath()
		if err != nil {
			return "", err
		}
		return filepath.Dir(exe), nil
	}
)

// Download fetches the desktop asset for tag, verifies it against the
// release's SHA256SUMS, stages it (chmod 0755) next to the running binary,
// and returns the staged path. The temp file is removed on any error.
func Download(ctx context.Context, tag string) (string, error) {
	asset, err := AssetName()
	if err != nil {
		return "", err
	}
	dir, err := stagingDir()
	if err != nil {
		return "", err
	}
	staged := filepath.Join(dir, "."+asset+".update")

	if err := fetchToFile(ctx, baseURL+tag+"/"+asset, staged); err != nil {
		return "", err
	}
	sums, err := fetchString(ctx, baseURL+tag+"/SHA256SUMS")
	if err != nil {
		_ = os.Remove(staged)
		return "", err
	}
	if err := VerifySHA256(staged, asset, sums); err != nil {
		_ = os.Remove(staged)
		return "", err
	}
	if err := os.Chmod(staged, 0o755); err != nil {
		_ = os.Remove(staged)
		return "", err
	}
	return staged, nil
}

func fetchToFile(ctx context.Context, url, dest string) error {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(dest)
		return err
	}
	return nil
}

func fetchString(ctx context.Context, url string) (string, error) {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return string(b), err
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "PgVoyager/"+version.Version)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("selfupdate: GET %s returned %d", url, resp.StatusCode)
	}
	return resp, nil
}
