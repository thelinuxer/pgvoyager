package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thelinuxer/pgvoyager/internal/version"
)

// hostAllowed restricts every update request — including each redirect hop —
// to GitHub-controlled hosts. GitHub release-asset downloads 302-redirect from
// github.com to GitHub's asset CDN (objects.githubusercontent.com /
// codeload.github.com), so the asset hosts must be included or downloads break.
// It is a seam so tests can point baseURL at a local httptest server.
var hostAllowed = defaultHostAllowed

func defaultHostAllowed(host string) bool {
	host = strings.ToLower(host)
	switch host {
	case "github.com", "api.github.com", "codeload.github.com":
		return true
	}
	return strings.HasSuffix(host, ".githubusercontent.com")
}

// Overridable for tests.
var (
	baseURL    = "https://github.com/" + version.GitHubRepo + "/releases/download/"
	httpClient = &http.Client{
		Timeout: 5 * time.Minute,
		// Follow redirects only to allowlisted GitHub hosts, and cap the chain.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("selfupdate: too many redirects")
			}
			if !hostAllowed(req.URL.Hostname()) {
				return fmt.Errorf("selfupdate: refusing redirect to disallowed host %q", req.URL.Host)
			}
			return nil
		},
	}
	// stagingDir returns where to stage the download. Prefer the running
	// executable's directory (so Apply can atomically rename); if that dir is
	// not writable, fall back to a user-private cache dir and let Apply elevate
	// the swap. The fallback must be user-owned and not world-writable: staging
	// in a shared /tmp path would let another local user swap the verified
	// binary before the privileged copy (TOCTOU → root code execution).
	stagingDir = func() (string, error) {
		exe, err := exePath()
		if err != nil {
			return "", err
		}
		dir := filepath.Dir(exe)
		if writableDir(dir) {
			return dir, nil
		}
		base, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		staging := filepath.Join(base, "pgvoyager", "update")
		if err := os.MkdirAll(staging, 0o700); err != nil {
			return "", err
		}
		return staging, nil
	}
)

// Download fetches the desktop asset for tag, verifies it against the
// release's SHA256SUMS, stages it (chmod 0755) next to the running binary,
// and returns the staged path. The temp file is removed on any error.
// onProgress, if non-nil, is called with the download percentage (0-100) as
// the binary streams; it is not called when the server omits Content-Length.
func Download(ctx context.Context, tag string, onProgress func(percent int)) (string, error) {
	asset, err := AssetName()
	if err != nil {
		return "", err
	}
	dir, err := stagingDir()
	if err != nil {
		return "", err
	}
	staged := filepath.Join(dir, "."+asset+".update")

	if err := fetchToFile(ctx, baseURL+tag+"/"+asset, staged, onProgress); err != nil {
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

const maxBinaryBytes = 256 << 20 // 256 MiB — generous for a Go binary

// progressWriter counts bytes written and reports integer percent (0-100) via
// cb, but only when the total size is known and the percent actually changes.
type progressWriter struct {
	total int64
	done  int64
	last  int
	cb    func(int)
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.done += int64(n)
	if w.cb != nil && w.total > 0 {
		pct := int(w.done * 100 / w.total)
		if pct > 100 {
			pct = 100
		}
		if pct != w.last {
			w.last = pct
			w.cb(pct)
		}
	}
	return n, nil
}

func fetchToFile(ctx context.Context, url, dest string, onProgress func(int)) error {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	dst := io.Writer(f)
	if onProgress != nil {
		dst = io.MultiWriter(f, &progressWriter{total: resp.ContentLength, cb: onProgress})
	}
	_, copyErr := io.Copy(dst, io.LimitReader(resp.Body, maxBinaryBytes))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return closeErr
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
	if !hostAllowed(req.URL.Hostname()) {
		return nil, fmt.Errorf("selfupdate: refusing request to disallowed host %q", req.URL.Host)
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
