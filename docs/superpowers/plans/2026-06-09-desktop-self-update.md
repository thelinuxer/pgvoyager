# Desktop Self-Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the PgVoyager desktop app check for, download, SHA256-verify, and self-apply new releases from its own Go process, with the frontend showing status and a "Restart now" button.

**Architecture:** A new `internal/selfupdate` package holds the mechanism (asset naming, checksum verify, writability preflight, download, atomic swap + relaunch) and a `Manager` state machine. The desktop `main` starts the `Manager` on a ticker and registers a desktop-only guarded `POST /update/restart`. A shared `GET /update/status` exposes state to the frontend. The headless server binary keeps the existing frontend-driven check.

**Tech Stack:** Go (stdlib `crypto/sha256`, `net/http`, `os/exec`, `runtime`), gin, SvelteKit/Svelte 5, GitHub Actions, bash installer.

**Spec:** `docs/superpowers/specs/2026-06-09-desktop-self-update-design.md`

---

## File Structure

- `backend/internal/version/version.go` — add `Edition` var + `IsDesktop()`. (modify)
- `backend/internal/selfupdate/asset.go` — `AssetName()`. (create)
- `backend/internal/selfupdate/checksum.go` — SHA256 parse + verify. (create)
- `backend/internal/selfupdate/fs.go` — `Writable()`, exe path helper. (create)
- `backend/internal/selfupdate/download.go` — `Download()`. (create)
- `backend/internal/selfupdate/apply.go` — `Apply()` + injectable spawn/signal. (create)
- `backend/internal/selfupdate/spawn_unix.go` / `spawn_windows.go` — platform spawn. (create)
- `backend/internal/selfupdate/manager.go` — `Manager` state machine. (create)
- `backend/internal/selfupdate/*_test.go` — unit tests. (create)
- `backend/internal/handlers/update.go` — add status + restart handlers, manager injection. (modify)
- `backend/internal/handlers/update_test.go` — handler tests. (create)
- `backend/internal/api/routes.go` — register `/update/status`. (modify)
- `backend/cmd/desktop/main.go` — construct/start Manager, wire status + restart route. (modify)
- `.github/workflows/release.yml` — desktop Edition ldflag + SHA256SUMS asset. (modify)
- `Makefile` — desktop/desktop-dev Edition ldflag. (modify)
- `frontend/src/lib/api/client.ts` — `UpdateStatus` type + `updateApi.status/restart`. (modify)
- `frontend/src/lib/components/Header.svelte` — poll status, render states, restart button. (modify)
- `frontend/src/lib/components/Header.svelte` styles — states. (modify)
- `packaging/linux/install.sh` — `--user` / no-sudo install. (modify)
- `README.md` — document `./install.sh --user`. (modify)
- `e2e/tests/tier1-critical/update-banner.spec.ts` — mocked status render. (create)

---

## Task 1: Edition build tag

**Files:**
- Modify: `backend/internal/version/version.go`
- Test: `backend/internal/version/version_test.go`

- [ ] **Step 1: Write the failing test**

```go
package version

import "testing"

func TestIsDesktopDefault(t *testing.T) {
	// Default build (no ldflag) is not the desktop edition.
	if IsDesktop() {
		t.Fatalf("IsDesktop() = true for default build, want false")
	}
}

func TestIsDesktopWhenSet(t *testing.T) {
	orig := Edition
	t.Cleanup(func() { Edition = orig })
	Edition = "desktop"
	if !IsDesktop() {
		t.Fatalf("IsDesktop() = false when Edition=desktop, want true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/version/ -run TestIsDesktop -v`
Expected: FAIL — `undefined: IsDesktop` / `undefined: Edition`.

- [ ] **Step 3: Add the var and helper**

Append to `backend/internal/version/version.go`:

```go
// Edition is set at build time via ldflags ("desktop" for the desktop
// wrapper, empty otherwise). It gates self-update behavior.
var Edition = ""

// IsDesktop reports whether this build is the desktop edition.
func IsDesktop() bool {
	return Edition == "desktop"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/version/ -run TestIsDesktop -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/version/version.go backend/internal/version/version_test.go
git commit -m "feat(version): add Edition build tag and IsDesktop helper"
```

---

## Task 2: selfupdate.AssetName

**Files:**
- Create: `backend/internal/selfupdate/asset.go`
- Test: `backend/internal/selfupdate/asset_test.go`

- [ ] **Step 1: Write the failing test**

```go
package selfupdate

import (
	"runtime"
	"testing"
)

func TestAssetNameCurrentPlatform(t *testing.T) {
	name, err := AssetName()
	if err != nil {
		t.Fatalf("AssetName() error: %v", err)
	}
	want := "pgvoyager-desktop-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if name != want {
		t.Fatalf("AssetName() = %q, want %q", name, want)
	}
}

func TestAssetNameForExplicit(t *testing.T) {
	got, err := assetNameFor("linux", "amd64")
	if err != nil || got != "pgvoyager-desktop-linux-amd64" {
		t.Fatalf("assetNameFor(linux,amd64) = %q, %v", got, err)
	}
	got, err = assetNameFor("windows", "amd64")
	if err != nil || got != "pgvoyager-desktop-windows-amd64.exe" {
		t.Fatalf("assetNameFor(windows,amd64) = %q, %v", got, err)
	}
	if _, err := assetNameFor("plan9", "amd64"); err == nil {
		t.Fatalf("assetNameFor(plan9) expected error, got nil")
	}
	if _, err := assetNameFor("linux", "mips"); err == nil {
		t.Fatalf("assetNameFor(mips) expected error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/selfupdate/ -run TestAssetName -v`
Expected: FAIL — package/functions undefined.

- [ ] **Step 3: Implement**

`backend/internal/selfupdate/asset.go`:

```go
// Package selfupdate implements desktop-edition in-place updates: resolving
// the current platform's release asset, verifying its SHA256, and atomically
// replacing the running executable.
package selfupdate

import (
	"fmt"
	"runtime"
)

// AssetName returns the GitHub release asset filename for the desktop binary
// on the current platform (e.g. "pgvoyager-desktop-linux-amd64").
func AssetName() (string, error) {
	return assetNameFor(runtime.GOOS, runtime.GOARCH)
}

func assetNameFor(goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return "", fmt.Errorf("selfupdate: unsupported OS %q", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("selfupdate: unsupported architecture %q", goarch)
	}
	name := fmt.Sprintf("pgvoyager-desktop-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/selfupdate/ -run TestAssetName -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/selfupdate/asset.go backend/internal/selfupdate/asset_test.go
git commit -m "feat(selfupdate): resolve platform release asset name"
```

---

## Task 3: SHA256 parse + verify

**Files:**
- Create: `backend/internal/selfupdate/checksum.go`
- Test: `backend/internal/selfupdate/checksum_test.go`

- [ ] **Step 1: Write the failing test**

```go
package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
)

const sums = "" +
	"aaa111  pgvoyager-desktop-linux-arm64\n" +
	"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  pgvoyager-desktop-linux-amd64\n"

func TestSHA256FromSums(t *testing.T) {
	got, err := sha256FromSums(sums, "pgvoyager-desktop-linux-amd64")
	if err != nil || got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("sha256FromSums = %q, %v", got, err)
	}
	if _, err := sha256FromSums(sums, "missing"); err == nil {
		t.Fatalf("expected error for missing asset")
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "blob")
	// Empty file → known SHA256 e3b0c442...
	if err := os.WriteFile(p, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(p, "pgvoyager-desktop-linux-amd64", sums); err != nil {
		t.Fatalf("VerifySHA256 (good) error: %v", err)
	}
	if err := os.WriteFile(p, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(p, "pgvoyager-desktop-linux-amd64", sums); err == nil {
		t.Fatalf("VerifySHA256 (bad) expected mismatch error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/selfupdate/ -run "TestSHA256FromSums|TestVerifySHA256" -v`
Expected: FAIL — functions undefined.

- [ ] **Step 3: Implement**

`backend/internal/selfupdate/checksum.go`:

```go
package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// VerifySHA256 returns nil iff the file at path hashes to the value recorded
// for assetName in a sha256sum-format sumsContent.
func VerifySHA256(path, assetName, sumsContent string) error {
	want, err := sha256FromSums(sumsContent, assetName)
	if err != nil {
		return err
	}
	got, err := sha256File(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("selfupdate: checksum mismatch for %s (got %s, want %s)", assetName, got, want)
	}
	return nil
}

// sha256FromSums finds the hash for assetName in `<hash>  <name>` lines.
func sha256FromSums(content, assetName string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[len(fields)-1] == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("selfupdate: no checksum found for %s", assetName)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/selfupdate/ -run "TestSHA256FromSums|TestVerifySHA256" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/selfupdate/checksum.go backend/internal/selfupdate/checksum_test.go
git commit -m "feat(selfupdate): SHA256SUMS parsing and file verification"
```

---

## Task 4: Writability preflight + exe path helper

**Files:**
- Create: `backend/internal/selfupdate/fs.go`
- Test: `backend/internal/selfupdate/fs_test.go`

- [ ] **Step 1: Write the failing test**

```go
package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWritableDirTrue(t *testing.T) {
	dir := t.TempDir()
	if !writableDir(dir) {
		t.Fatalf("writableDir(%s) = false, want true", dir)
	}
}

func TestWritableDirFalse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based read-only dir not reliable on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	dir := t.TempDir()
	ro := filepath.Join(dir, "ro")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatal(err)
	}
	if writableDir(ro) {
		t.Fatalf("writableDir(read-only) = true, want false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/selfupdate/ -run TestWritableDir -v`
Expected: FAIL — `writableDir` undefined.

- [ ] **Step 3: Implement**

`backend/internal/selfupdate/fs.go`:

```go
package selfupdate

import (
	"os"
	"path/filepath"
)

// exePath returns the absolute, symlink-resolved path of the running binary.
func exePath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved, nil
	}
	return p, nil
}

// Writable reports whether the running executable's directory can be written
// by the current user (a precondition for in-place self-replace).
func Writable() bool {
	exe, err := exePath()
	if err != nil {
		return false
	}
	return writableDir(filepath.Dir(exe))
}

func writableDir(dir string) bool {
	f, err := os.CreateTemp(dir, ".pgvoyager-wtest-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/selfupdate/ -run TestWritableDir -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/selfupdate/fs.go backend/internal/selfupdate/fs_test.go
git commit -m "feat(selfupdate): executable path helper and writability preflight"
```

---

## Task 5: Download (asset + checksum, verified, staged)

**Files:**
- Create: `backend/internal/selfupdate/download.go`
- Test: `backend/internal/selfupdate/download_test.go`

- [ ] **Step 1: Write the failing test**

```go
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

	// Point download at the fake release host + a temp staging dir.
	origBase, origDir := baseURL, stagingDir
	t.Cleanup(func() { baseURL, stagingDir = origBase, origDir })
	baseURL = srv.URL + "/"
	dir := t.TempDir()
	stagingDir = func() (string, error) { return dir, nil }

	staged, err := Download(context.Background(), "v9.9.9")
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

	origBase, origDir := baseURL, stagingDir
	t.Cleanup(func() { baseURL, stagingDir = origBase, origDir })
	baseURL = srv.URL + "/"
	dir := t.TempDir()
	stagingDir = func() (string, error) { return dir, nil }

	if _, err := Download(context.Background(), "v9.9.9"); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("staged temp not cleaned up: %v", entries)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/selfupdate/ -run TestDownload -v`
Expected: FAIL — `Download`, `baseURL`, `stagingDir` undefined.

- [ ] **Step 3: Implement**

`backend/internal/selfupdate/download.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/selfupdate/ -run TestDownload -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/selfupdate/download.go backend/internal/selfupdate/download_test.go
git commit -m "feat(selfupdate): download + verify + stage release asset"
```

---

## Task 6: Apply (atomic swap + relaunch), platform spawn

**Files:**
- Create: `backend/internal/selfupdate/apply.go`
- Create: `backend/internal/selfupdate/spawn_unix.go`
- Create: `backend/internal/selfupdate/spawn_windows.go`
- Test: `backend/internal/selfupdate/apply_test.go`

- [ ] **Step 1: Write the failing test**

```go
package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplySwapsAndSpawns(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pgvoyager-desktop")
	if err := os.WriteFile(exe, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, ".staged")
	if err := os.WriteFile(staged, []byte("NEW"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Stub out the OS-touching steps.
	origExe, origSpawn, origSignal := exePathFn, spawnDetached, signalSelf
	t.Cleanup(func() { exePathFn, spawnDetached, signalSelf = origExe, origSpawn, origSignal })
	exePathFn = func() (string, error) { return exe, nil }
	var spawned string
	spawnDetached = func(p string) error { spawned = p; return nil }
	signaled := false
	signalSelf = func() error { signaled = true; return nil }

	if err := Apply(staged); err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "NEW" {
		t.Fatalf("exe content = %q, want NEW", got)
	}
	if spawned != exe {
		t.Fatalf("spawned %q, want %q", spawned, exe)
	}
	if !signaled {
		t.Fatalf("signalSelf not called")
	}
}

func TestApplyDoesNotSignalWhenSpawnFails(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "pgvoyager-desktop")
	_ = os.WriteFile(exe, []byte("OLD"), 0o755)
	staged := filepath.Join(dir, ".staged")
	_ = os.WriteFile(staged, []byte("NEW"), 0o755)

	origExe, origSpawn, origSignal := exePathFn, spawnDetached, signalSelf
	t.Cleanup(func() { exePathFn, spawnDetached, signalSelf = origExe, origSpawn, origSignal })
	exePathFn = func() (string, error) { return exe, nil }
	spawnDetached = func(p string) error { return os.ErrPermission }
	signaled := false
	signalSelf = func() error { signaled = true; return nil }

	if err := Apply(staged); err == nil {
		t.Fatalf("expected error when spawn fails")
	}
	if signaled {
		t.Fatalf("signalSelf must not run when spawn fails")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/selfupdate/ -run TestApply -v`
Expected: FAIL — `Apply`, `exePathFn`, `spawnDetached`, `signalSelf` undefined.

- [ ] **Step 3: Implement apply.go**

`backend/internal/selfupdate/apply.go`:

```go
package selfupdate

import (
	"fmt"
	"os"
)

// Injectable seams for testing.
var (
	exePathFn     = exePath
	spawnDetached = realSpawnDetached
	signalSelf    = realSignalSelf
)

// Apply replaces the running executable with the staged binary and relaunches
// it. It renames the staged file over the current executable (atomic on the
// same filesystem; permitted over a running binary on Linux/macOS), spawns the
// new binary detached, and only then signals the current process to exit. If
// the spawn fails the current process is left running so the user can retry.
func Apply(stagedPath string) error {
	exe, err := exePathFn()
	if err != nil {
		return err
	}
	if err := os.Rename(stagedPath, exe); err != nil {
		return fmt.Errorf("selfupdate: swap binary: %w", err)
	}
	if err := spawnDetached(exe); err != nil {
		return fmt.Errorf("selfupdate: relaunch: %w", err)
	}
	return signalSelf()
}

func realSignalSelf() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return p.Signal(os.Interrupt) // desktop main bridges Interrupt/SIGTERM → shutdown
}
```

- [ ] **Step 4: Implement platform spawn files**

`backend/internal/selfupdate/spawn_unix.go`:

```go
//go:build !windows

package selfupdate

import (
	"os/exec"
	"strings"
	"syscall"
)

// realSpawnDetached starts the new binary in its own process group so it
// survives the current process exiting, with a clean port so it binds fresh.
func realSpawnDetached(exe string) error {
	cmd := exec.Command(exe)
	cmd.Env = cleanEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

func cleanEnv() []string {
	out := make([]string, 0, len(osEnviron()))
	for _, kv := range osEnviron() {
		if strings.HasPrefix(kv, "PGVOYAGER_PORT=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
```

`backend/internal/selfupdate/spawn_windows.go`:

```go
//go:build windows

package selfupdate

import (
	"os/exec"
	"strings"
)

func realSpawnDetached(exe string) error {
	cmd := exec.Command(exe)
	cmd.Env = cleanEnv()
	return cmd.Start()
}

func cleanEnv() []string {
	out := make([]string, 0, len(osEnviron()))
	for _, kv := range osEnviron() {
		if strings.HasPrefix(kv, "PGVOYAGER_PORT=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}
```

Add the `osEnviron` seam at the bottom of `apply.go` (shared, testable):

```go
var osEnviron = os.Environ
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend && go test ./internal/selfupdate/ -run TestApply -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/selfupdate/apply.go backend/internal/selfupdate/spawn_unix.go backend/internal/selfupdate/spawn_windows.go backend/internal/selfupdate/apply_test.go
git commit -m "feat(selfupdate): atomic swap + detached relaunch with platform spawn"
```

---

## Task 7: Manager state machine

**Files:**
- Create: `backend/internal/selfupdate/manager.go`
- Test: `backend/internal/selfupdate/manager_test.go`

- [ ] **Step 1: Write the failing test**

```go
package selfupdate

import (
	"context"
	"errors"
	"testing"
)

func newTestManager() *Manager {
	m := NewManager("1.0.0")
	return m
}

func TestManagerNoUpdate(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v1.0.0", "url", nil }
	m.cycle(context.Background())
	if got := m.Status().Status; got != StatusIdle {
		t.Fatalf("status = %q, want idle", got)
	}
}

func TestManagerDownloadsWhenNewer(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "url", nil }
	m.writable = func() bool { return true }
	m.downloadFn = func(context.Context, string) (string, error) { return "/tmp/staged", nil }
	m.cycle(context.Background())
	st := m.Status()
	if st.Status != StatusReady {
		t.Fatalf("status = %q, want ready", st.Status)
	}
	if st.LatestVersion != "2.0.0" {
		t.Fatalf("latest = %q, want 2.0.0", st.LatestVersion)
	}
	if m.staged != "/tmp/staged" {
		t.Fatalf("staged = %q", m.staged)
	}
}

func TestManagerManualWhenNotWritable(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "rel-url", nil }
	m.writable = func() bool { return false }
	called := false
	m.downloadFn = func(context.Context, string) (string, error) { called = true; return "", nil }
	m.cycle(context.Background())
	if m.Status().Status != StatusManual {
		t.Fatalf("status = %q, want manual", m.Status().Status)
	}
	if called {
		t.Fatalf("downloadFn must not run when not writable")
	}
}

func TestManagerErrorOnDownloadFailure(t *testing.T) {
	m := newTestManager()
	m.fetchLatest = func(context.Context) (string, string, error) { return "v2.0.0", "url", nil }
	m.writable = func() bool { return true }
	m.downloadFn = func(context.Context, string) (string, error) { return "", errors.New("boom") }
	m.cycle(context.Background())
	if m.Status().Status != StatusError {
		t.Fatalf("status = %q, want error", m.Status().Status)
	}
}

func TestManagerRestartGuards(t *testing.T) {
	m := newTestManager()
	if err := m.Restart(); err == nil {
		t.Fatalf("Restart with no staged update should error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/selfupdate/ -run TestManager -v`
Expected: FAIL — `Manager`, `NewManager`, statuses undefined.

- [ ] **Step 3: Implement**

`backend/internal/selfupdate/manager.go`:

```go
package selfupdate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Status is the coarse self-update state surfaced to the frontend.
type Status string

const (
	StatusIdle        Status = "idle"
	StatusChecking    Status = "checking"
	StatusDownloading Status = "downloading"
	StatusReady       Status = "ready"
	StatusError       Status = "error"
	StatusManual      Status = "manual"
)

// State is a snapshot of the manager for the status endpoint.
type State struct {
	Edition        string `json:"edition"`
	Status         Status `json:"status"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseURL     string `json:"releaseUrl"`
	Error          string `json:"error,omitempty"`
}

// Manager owns the desktop self-update lifecycle: periodic check, background
// download+verify, and applying a staged update on request.
type Manager struct {
	mu      sync.Mutex
	state   State
	staged  string
	current string

	// Seams (overridable in tests).
	fetchLatest func(context.Context) (tag, htmlURL string, err error)
	downloadFn  func(context.Context, string) (string, error)
	writable    func() bool
	applyFn     func(string) error
}

// NewManager builds a Manager wired to the real fetch/download/apply functions.
func NewManager(currentVersion string) *Manager {
	m := &Manager{
		current:     currentVersion,
		fetchLatest: fetchLatestRelease,
		downloadFn:  Download,
		writable:    Writable,
		applyFn:     Apply,
	}
	m.state = State{
		Edition:        "desktop",
		Status:         StatusIdle,
		CurrentVersion: currentVersion,
	}
	return m
}

// Start runs an immediate check then re-checks every interval until ctx ends.
func (m *Manager) Start(ctx context.Context, interval time.Duration) {
	go func() {
		m.cycle(ctx)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.cycle(ctx)
			}
		}
	}()
}

func (m *Manager) setStatus(s Status, mutate func(*State)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Status = s
	if mutate != nil {
		mutate(&m.state)
	}
}

// cycle performs one check→(download) pass. Once ready, it stops re-checking
// effectively by short-circuiting (a staged update remains ready).
func (m *Manager) cycle(ctx context.Context) {
	m.mu.Lock()
	already := m.state.Status == StatusReady
	m.mu.Unlock()
	if already {
		return
	}

	m.setStatus(StatusChecking, nil)
	tag, htmlURL, err := m.fetchLatest(ctx)
	if err != nil {
		m.setStatus(StatusError, func(s *State) { s.Error = err.Error() })
		return
	}
	latest := strings.TrimPrefix(tag, "v")
	current := strings.TrimPrefix(m.current, "v")
	if current == "dev" || compareVersions(current, latest) >= 0 {
		m.setStatus(StatusIdle, func(s *State) { s.LatestVersion = latest; s.ReleaseURL = htmlURL; s.Error = "" })
		return
	}

	if !m.writable() {
		m.setStatus(StatusManual, func(s *State) { s.LatestVersion = latest; s.ReleaseURL = htmlURL; s.Error = "" })
		return
	}

	m.setStatus(StatusDownloading, func(s *State) { s.LatestVersion = latest; s.ReleaseURL = htmlURL })
	staged, err := m.downloadFn(ctx, tag)
	if err != nil {
		m.setStatus(StatusError, func(s *State) { s.Error = err.Error() })
		return
	}
	m.mu.Lock()
	m.staged = staged
	m.state.Status = StatusReady
	m.state.Error = ""
	m.mu.Unlock()
}

// Status returns a snapshot.
func (m *Manager) Status() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Restart applies a previously-staged, verified update and relaunches.
func (m *Manager) Restart() error {
	m.mu.Lock()
	ready := m.state.Status == StatusReady && m.staged != ""
	staged := m.staged
	m.mu.Unlock()
	if !ready {
		return fmt.Errorf("selfupdate: no staged update to apply")
	}
	return m.applyFn(staged)
}
```

> Note: `fetchLatestRelease` and `compareVersions` already exist in
> `internal/handlers/update.go`. Move them into this package in Task 8 so
> `manager.go` compiles. Until then, this task's test file will not compile on
> its own — Task 8 completes the wiring. If implementing strictly task-by-task,
> do Task 8's "Step 3a" (move helpers) before running Task 7's tests.

- [ ] **Step 3a (prerequisite): move shared helpers into selfupdate**

Create `backend/internal/selfupdate/github.go` and move `GitHubRelease`,
`fetchLatestRelease`, and `compareVersions` here (verbatim from
`internal/handlers/update.go`), adapted to return `(tag, htmlURL string, err error)`:

```go
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/thelinuxer/pgvoyager/internal/version"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

var latestReleaseURL = "https://api.github.com/repos/" + version.GitHubRepo + "/releases/latest"

// fetchLatestRelease returns the latest release tag and its HTML URL.
func fetchLatestRelease(ctx context.Context) (string, string, error) {
	resp, err := httpGet(ctx, latestReleaseURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", err
	}
	if rel.TagName == "" {
		return "", "", fmt.Errorf("selfupdate: empty tag in latest release")
	}
	return rel.TagName, rel.HTMLURL, nil
}

// compareVersions compares dotted numeric versions: -1 if a<b, 0 equal, 1 a>b.
func compareVersions(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		var na, nb int
		fmt.Sscanf(pa[i], "%d", &na)
		fmt.Sscanf(pb[i], "%d", &nb)
		if na < nb {
			return -1
		}
		if na > nb {
			return 1
		}
	}
	switch {
	case len(pa) < len(pb):
		return -1
	case len(pa) > len(pb):
		return 1
	default:
		return 0
	}
}
```

`httpGet` is shared from `download.go` (same package). `fetchLatestRelease`
takes `context.Context` and reuses it.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/selfupdate/ -v`
Expected: PASS (all selfupdate tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/selfupdate/manager.go backend/internal/selfupdate/github.go backend/internal/selfupdate/manager_test.go
git commit -m "feat(selfupdate): manager state machine + github helpers"
```

---

## Task 8: Handlers — status + restart, manager injection

**Files:**
- Modify: `backend/internal/handlers/update.go`
- Test: `backend/internal/handlers/update_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/handlers/ -run TestUpdate -v`
Expected: FAIL — `SetUpdateManager`, `UpdateStatus`, `UpdateRestart` undefined.

- [ ] **Step 3: Implement**

In `backend/internal/handlers/update.go`:

1. Remove the now-moved helpers `fetchLatestRelease` and `compareVersions`
   (they live in `internal/selfupdate`). Keep `CheckUpdate`/`GetVersion` but
   reimplement `CheckUpdate`'s fetch using a small local call OR keep its own
   copy. To avoid a second GitHub fetcher, change `CheckUpdate` to build its
   response from a one-shot `selfupdate`-style check. Simplest: keep
   `CheckUpdate` calling a thin local fetch retained in this file. **Decision:**
   keep `CheckUpdate` exactly as-is but have it call the kept local helpers
   `fetchLatestReleaseLegacy`/`compareVersionsLegacy` (rename the existing ones
   in this file with a `Legacy` suffix so they don't clash). This preserves the
   server-edition `/update/check` behavior unchanged.

2. Add the manager seam + new handlers:

```go
import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thelinuxer/pgvoyager/internal/selfupdate"
	"github.com/thelinuxer/pgvoyager/internal/version"
)

// updateManager is set by the desktop binary; nil for the server edition.
var updateManager *selfupdate.Manager

// SetUpdateManager wires the desktop self-update manager into the handlers.
func SetUpdateManager(m *selfupdate.Manager) { updateManager = m }

// UpdateStatus returns the current self-update state. Desktop edition reports
// the live manager state; server edition reports a computed check result.
func UpdateStatus(c *gin.Context) {
	if updateManager != nil {
		c.JSON(http.StatusOK, updateManager.Status())
		return
	}
	// Server edition: reuse the existing check to report hasUpdate as "manual".
	resp := computeServerStatus()
	c.JSON(http.StatusOK, resp)
}

func computeServerStatus() gin.H {
	rel, err := fetchLatestReleaseLegacy()
	current := version.Version
	if err != nil {
		return gin.H{"edition": "server", "status": "idle", "currentVersion": current, "latestVersion": current, "releaseUrl": version.ReleasesURL()}
	}
	resp := buildUpdateResponse(current, rel)
	status := "idle"
	if resp.HasUpdate {
		status = "manual"
	}
	return gin.H{
		"edition":        "server",
		"status":         status,
		"currentVersion": resp.CurrentVersion,
		"latestVersion":  resp.LatestVersion,
		"releaseUrl":     resp.ReleaseURL,
	}
}

// UpdateRestart applies a staged update (desktop edition only).
func UpdateRestart(c *gin.Context) {
	if updateManager == nil || !version.IsDesktop() {
		c.JSON(http.StatusConflict, gin.H{"error": "self-update not supported for this build"})
		return
	}
	if err := updateManager.Restart(); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"restarting": true})
	// Restart() already triggered apply (spawn + self-signal); the short
	// shutdown delay is owned by the desktop main's signal handler.
	_ = time.Second
}
```

> `buildUpdateResponse` stays in this file (used by both `CheckUpdate` and
> `computeServerStatus`). Rename existing `fetchLatestRelease` →
> `fetchLatestReleaseLegacy` and `compareVersions` → keep only if
> `buildUpdateResponse` needs it (it does) → rename to `compareVersionsLegacy`
> and update `buildUpdateResponse` to call it.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/handlers/ -run TestUpdate -v`
Expected: PASS

- [ ] **Step 5: Run the whole backend build + tests**

Run: `cd backend && go build ./... && go test ./internal/... `
Expected: build OK, tests PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handlers/update.go backend/internal/handlers/update_test.go
git commit -m "feat(handlers): update status + desktop-only restart, manager injection"
```

---

## Task 9: Routes + desktop wiring

**Files:**
- Modify: `backend/internal/api/routes.go:114-116`
- Modify: `backend/cmd/desktop/main.go`

- [ ] **Step 1: Register the shared status route**

In `backend/internal/api/routes.go`, change the update block:

```go
		// Version and updates
		api.GET("/version", handlers.GetVersion)
		api.GET("/update/check", handlers.CheckUpdate)
		api.GET("/update/status", handlers.UpdateStatus)
```

(The desktop-only `/update/restart` is registered in `cmd/desktop`, not here,
so the server binary never exposes it.)

- [ ] **Step 2: Wire the manager + restart route in desktop main**

In `backend/cmd/desktop/main.go`:

1. Add imports: `"github.com/thelinuxer/pgvoyager/internal/handlers"`,
   `"github.com/thelinuxer/pgvoyager/internal/selfupdate"`,
   `"github.com/thelinuxer/pgvoyager/internal/version"`.

2. In `buildRouter()`, after `api.RegisterRoutes(r)`, add:

```go
	// Desktop-only: the mutating restart route lives here so the headless
	// server binary never exposes "replace my binary" over HTTP.
	r.POST("/api/update/restart", handlers.UpdateRestart)
```

3. In `main()`, after the `ctx, cancel := context.WithCancel(...)` line and
   before `chromelaunch.Run`, construct and start the manager and inject it:

```go
	updater := selfupdate.NewManager(version.Version)
	handlers.SetUpdateManager(updater)
	updater.Start(ctx, 6*time.Hour)
```

- [ ] **Step 3: Build the desktop binary**

Run: `cd backend && go build ./cmd/desktop`
Expected: builds with no errors.

- [ ] **Step 4: Run full backend tests**

Run: `cd backend && go build ./... && go test ./internal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/routes.go backend/cmd/desktop/main.go
git commit -m "feat: register update status route and wire desktop self-update manager"
```

---

## Task 10: Makefile — desktop Edition ldflag

**Files:**
- Modify: `Makefile:148-151` (the `desktop` and `desktop-dev` targets)

- [ ] **Step 1: Read current targets**

Run: `grep -nA2 "^desktop:" Makefile; grep -nA2 "^desktop-dev:" Makefile`

- [ ] **Step 2: Add the Edition ldflag**

Add near the top with the other vars (after `LDFLAGS :=` line):

```make
DESKTOP_LDFLAGS := $(LDFLAGS) -X github.com/thelinuxer/pgvoyager/internal/version.Edition=desktop
```

Change the `desktop` and `desktop-dev` recipes to use `$(DESKTOP_LDFLAGS)`:

```make
desktop: build-frontend-prod
	cd backend && go build -ldflags="$(DESKTOP_LDFLAGS)" -o ../bin/pgvoyager-desktop ./cmd/desktop
```

(Apply the same `-ldflags="$(DESKTOP_LDFLAGS)"` to `desktop-dev`'s build line.)

- [ ] **Step 3: Verify the ldflag is applied**

Run:
```bash
make desktop
cd backend && go build -ldflags="-X github.com/thelinuxer/pgvoyager/internal/version.Edition=desktop" -o /tmp/pv-desktop ./cmd/desktop && echo OK
```
Expected: build succeeds. (Edition is internal; correctness is exercised by the
manual verification in Task 17.)

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "build(make): tag desktop builds with version.Edition=desktop"
```

---

## Task 11: release.yml — Edition ldflag + SHA256SUMS

**Files:**
- Modify: `.github/workflows/release.yml` (desktop build commands; release files list)

- [ ] **Step 1: Add a desktop LDFLAGS var and use it for desktop builds**

In the `Build binaries` step's script, before the desktop builds, add:

```bash
          DESKTOP_LDFLAGS="${LDFLAGS} -X github.com/thelinuxer/pgvoyager/internal/version.Edition=desktop"
```

Change each `./cmd/desktop` build line to use `-ldflags="${DESKTOP_LDFLAGS}"`
instead of `-ldflags="${LDFLAGS}"`. (Five lines: linux amd64/arm64, darwin
amd64/arm64, windows amd64.)

- [ ] **Step 2: Generate SHA256SUMS in the release job**

In the `release` job, after the `Download build artifacts (raw binaries)` step
and before `Create Release`, add:

```yaml
      - name: Generate SHA256SUMS
        run: |
          cd releases
          sha256sum pgvoyager-desktop-* > SHA256SUMS
          cat SHA256SUMS
```

- [ ] **Step 3: Add SHA256SUMS to the release files list**

In the `Create Release` step's `files:` list, add a line:

```yaml
            releases/SHA256SUMS
```

- [ ] **Step 4: Validate workflow YAML**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/release.yml')); print('YAML OK')"`
Expected: `YAML OK`.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci(release): tag desktop Edition and publish SHA256SUMS"
```

---

## Task 12: Frontend API client

**Files:**
- Modify: `frontend/src/lib/api/client.ts:340-351`

- [ ] **Step 1: Add the status type + API methods**

Replace the `UpdateCheckResponse`/`updateApi` block with:

```ts
export interface UpdateCheckResponse {
	currentVersion: string;
	latestVersion: string;
	hasUpdate: boolean;
	releaseUrl: string;
}

export type UpdateStatusValue =
	| 'idle'
	| 'checking'
	| 'downloading'
	| 'ready'
	| 'error'
	| 'manual';

export interface UpdateStatus {
	edition: 'desktop' | 'server';
	status: UpdateStatusValue;
	currentVersion: string;
	latestVersion: string;
	releaseUrl: string;
	error?: string;
}

export const updateApi = {
	getVersion: () => fetchAPI<VersionResponse>('/version'),
	checkUpdate: () => fetchAPI<UpdateCheckResponse>('/update/check'),
	status: () => fetchAPI<UpdateStatus>('/update/status'),
	restart: () => fetchAPI<{ restarting: boolean }>('/update/restart', { method: 'POST' })
};
```

- [ ] **Step 2: Type-check**

Run: `cd frontend && npx svelte-check --threshold error --tsconfig ./tsconfig.json 2>&1 | grep -i "client.ts" || echo "client.ts: no errors"`
Expected: `client.ts: no errors`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/api/client.ts
git commit -m "feat(client): update status + restart API"
```

---

## Task 13: Header.svelte — status polling + restart UI

**Files:**
- Modify: `frontend/src/lib/components/Header.svelte` (script + version badge markup + styles)

- [ ] **Step 1: Replace the update script block**

Replace lines that import `UpdateCheckResponse` and the `updateInfo`/`checkForUpdate`/`onMount` block with:

```ts
	import { connectionApi, updateApi, type UpdateStatus } from '$lib/api/client';
	// ...keep other imports unchanged...

	let update = $state<UpdateStatus | null>(null);
	let restarting = $state(false);

	// Polling cadence: the desktop process does the heavy lifting (check +
	// download). The UI only polls to learn when an update is ready.
	const UPDATE_POLL_MS = 30 * 60 * 1000;

	async function refreshUpdateStatus() {
		try {
			update = await updateApi.status();
		} catch {
			// Non-fatal: badge falls back to nothing.
		}
	}

	async function handleRestart() {
		if (restarting) return;
		restarting = true;
		try {
			await updateApi.restart();
			// Backend swaps + relaunches; this window will be torn down.
		} catch {
			restarting = false;
		}
	}

	onMount(() => {
		refreshUpdateStatus();
		const timer = setInterval(refreshUpdateStatus, UPDATE_POLL_MS);
		return () => clearInterval(timer);
	});
```

(Keep the existing `handleConnect`/`handleDisconnect` functions unchanged.)

- [ ] **Step 2: Replace the version-badge markup**

Replace the `{#if updateInfo} … {/if}` block in the header markup with:

```svelte
		{#if update}
			{#if restarting || update.status === 'restarting'}
				<span class="version-badge" title="Updating…">
					<Icon name="refresh" size={12} class="spinning" />
					Updating…
				</span>
			{:else if update.status === 'ready'}
				<button class="version-badge update-ready" onclick={handleRestart}
				        title="Update {update.latestVersion} ready — restart to apply"
				        data-testid="btn-update-restart">
					<span class="update-dot"></span>
					Restart to update
				</button>
			{:else if update.status === 'downloading'}
				<span class="version-badge" title="Downloading update {update.latestVersion}…">
					<Icon name="refresh" size={12} class="spinning" />
					{update.currentVersion}
				</span>
			{:else if update.status === 'manual'}
				<a href={update.releaseUrl} target="_blank" rel="noopener noreferrer"
				   class="version-badge update-available"
				   title="Update available! Click to download {update.latestVersion}">
					<span class="update-dot"></span>
					{update.currentVersion}
				</a>
			{:else}
				<span class="version-badge" title="PgVoyager {update.currentVersion}">
					{update.currentVersion}
				</span>
			{/if}
		{/if}
```

- [ ] **Step 3: Add the ready-state style**

In the `<style>` block, after `.version-badge.update-available` rules, add:

```css
	.version-badge.update-ready {
		border: none;
		cursor: pointer;
		background: var(--color-primary);
		color: #fff;
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}

	.version-badge.update-ready:hover {
		filter: brightness(1.05);
	}
```

- [ ] **Step 4: Type-check**

Run: `cd frontend && npx svelte-check --threshold error --tsconfig ./tsconfig.json 2>&1 | grep -i "Header.svelte" || echo "Header.svelte: no errors"`
Expected: `Header.svelte: no errors`.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/components/Header.svelte
git commit -m "feat(header): poll update status, show Restart-to-update action"
```

---

## Task 14: Installer — user-writable install option

**Files:**
- Modify: `packaging/linux/install.sh`

- [ ] **Step 1: Parse a `--user` flag and pick install dir + sudo mode**

Replace the argument/INSTALL_DIR section (`PGVOYAGER_PORT="${1:-5137}"` …
`INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"`) with:

```bash
# Parse args: optional --user flag, optional port.
USER_INSTALL=0
PGVOYAGER_PORT="5137"
for arg in "$@"; do
    case "$arg" in
        --user) USER_INSTALL=1 ;;
        ''|*[!0-9]*) ;;            # ignore non-numeric args
        *) PGVOYAGER_PORT="$arg" ;;
    esac
done

if [ "$USER_INSTALL" -eq 1 ]; then
    INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
else
    INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
fi

# Use sudo only when the install dir is not writable by the current user.
mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ -w "$INSTALL_DIR" ]; then
    SUDO=""
else
    SUDO="sudo"
fi
```

- [ ] **Step 2: Route all privileged copies through `$SUDO`**

Replace every `sudo cp`/`sudo chmod`/`sudo pkill` in the script with
`${SUDO} cp`/`${SUDO} chmod`/`${SUDO} pkill`. (Lines that install pgvoyager,
pgvoyager-mcp, pgvoyager-desktop, pgvoyager-launcher, and the "Stopping running
instance" `sudo pkill`.)

- [ ] **Step 3: Warn if a user dir is not on PATH**

After the install dir is finalized (right after the `SUDO` block), add:

```bash
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        if [ "$USER_INSTALL" -eq 1 ]; then
            echo ""
            echo "  NOTE: $INSTALL_DIR is not on your PATH."
            echo "  Add this to your shell profile:"
            echo "      export PATH=\"$INSTALL_DIR:\$PATH\""
        fi
        ;;
esac
```

- [ ] **Step 4: Shellcheck / syntax check**

Run: `bash -n packaging/linux/install.sh && echo "syntax OK"`
Expected: `syntax OK`.

- [ ] **Step 5: Dry-run into a temp dir**

Run:
```bash
INSTALL_DIR="$(mktemp -d)/bin" bash packaging/linux/install.sh --user 5137 || true
```
Expected: installs without invoking sudo (no password prompt); prints success.
(Binaries may be absent in a source checkout — the script skips missing files.)

- [ ] **Step 6: Commit**

```bash
git add packaging/linux/install.sh
git commit -m "feat(installer): --user install to ~/.local/bin without sudo"
```

---

## Task 15: README — document user install

**Files:**
- Modify: `README.md` (Desktop App section added earlier)

- [ ] **Step 1: Add an auto-update note under the Desktop App section**

After the Linux desktop install block in README's Installation → Desktop App,
add:

```markdown
> **Auto-update:** install to a user-writable location to enable in-app
> updates — `./install.sh --user` installs to `~/.local/bin` (no sudo). The
> desktop app then checks for new releases, downloads them in the background,
> and offers a **"Restart to update"** button. Root-owned installs
> (`/usr/local/bin`) show a manual download link instead.
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: document desktop auto-update and --user install"
```

---

## Task 16: E2E — Restart-to-update renders on ready status

**Files:**
- Create: `e2e/tests/tier1-critical/update-banner.spec.ts`

- [ ] **Step 1: Write the test (mocks the status endpoint)**

```ts
import { test, expect } from '@playwright/test';

const BASE = process.env.BASE_URL || 'http://localhost:5137';

test.describe('Update banner', () => {
  test('shows "Restart to update" when status is ready', async ({ page }) => {
    // Intercept the status poll and force a ready desktop update.
    await page.route('**/api/update/status', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          edition: 'desktop',
          status: 'ready',
          currentVersion: '0.3.5',
          latestVersion: '9.9.9',
          releaseUrl: 'https://example.com'
        })
      })
    );

    await page.goto(BASE);
    await expect(page.locator('[data-testid="btn-update-restart"]')).toBeVisible({ timeout: 10000 });
  });
});
```

- [ ] **Step 2: Run it against the dev/prod app**

Run:
```bash
cd e2e && CI=true TEST_PG_USER=hms TEST_PG_PASSWORD=hms npx playwright test update-banner --project=chromium --reporter=line
```
Expected: 1 passed. (Build the prod binary first via `make build-frontend-prod && cd backend && go build -o ../pgvoyager ./cmd/server` so the served frontend includes the new Header.)

- [ ] **Step 3: Commit**

```bash
git add e2e/tests/tier1-critical/update-banner.spec.ts
git commit -m "test(e2e): update banner shows Restart-to-update on ready status"
```

---

## Task 17: Manual end-to-end verification

**Files:** none (manual)

- [ ] **Step 1: Build an "old" desktop binary into a user-writable dir**

```bash
make build-frontend-prod
mkdir -p ~/.local/bin
cd backend && go build \
  -ldflags="-X github.com/thelinuxer/pgvoyager/internal/version.Version=v0.0.1 -X github.com/thelinuxer/pgvoyager/internal/version.Edition=desktop" \
  -o ~/.local/bin/pgvoyager-desktop ./cmd/desktop
```

- [ ] **Step 2: Run it and observe auto-update**

```bash
~/.local/bin/pgvoyager-desktop
```
Expected: within the poll cycle (force a faster interval temporarily if needed),
the badge shows downloading → **"Restart to update"**. The staged file
`~/.local/bin/.pgvoyager-desktop-linux-amd64.update` appears.

- [ ] **Step 3: Click "Restart to update"**

Expected: window closes and relaunches; new window reports the latest version
(`/api/version`). `~/.local/bin/pgvoyager-desktop` now matches the latest
release binary.

- [ ] **Step 4: Verify root-owned fallback**

Run the existing `/usr/local/bin/pgvoyager-desktop`; confirm the badge shows the
manual release link (status `manual`), not the restart button.

- [ ] **Step 5: Document the result** in the PR description; no commit.

---

## Self-Review notes

- **Spec coverage:** edition tag (T1,T10,T11) · SHA256SUMS pipeline (T11) ·
  AssetName/Writable/Download/Apply/Manager (T2–T7) · status+restart routes,
  desktop-only restart (T8,T9) · frontend status poll + restart (T12,T13) ·
  installer `--user` (T14) · README (T15) · tests incl. E2E + manual (T2–T8,
  T16,T17). All spec sections mapped.
- **Type consistency:** `selfupdate.Manager`, `State`, `Status*` constants,
  `NewManager(string)`, `Start(ctx,interval)`, `Status()`, `Restart()` used
  consistently across T7–T9. Handler seam `SetUpdateManager`/`UpdateStatus`/
  `UpdateRestart` consistent T8–T9. Frontend `UpdateStatus`/`updateApi.status`/
  `updateApi.restart` consistent T12–T13,T16.
- **Helper move:** `fetchLatestRelease`/`compareVersions` move to `selfupdate`
  (T7 step 3a); handler keeps `*Legacy` copies for `/update/check`
  back-compat (T8 step 3). Do T7-3a before T7 tests if executing strictly
  in order.
