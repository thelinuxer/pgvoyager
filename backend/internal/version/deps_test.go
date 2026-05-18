package version

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

// TestGoToolchainAboveCVEFloor enforces a minimum Go toolchain version so
// dependency vulnerabilities flagged in phase 5 (TLS handshake, net/url
// DoS, html/template, etc.) can't silently regress via a `go.mod` rollback.
// The floor matches the lowest stdlib patch level that covered every
// govulncheck finding.
func TestGoToolchainAboveCVEFloor(t *testing.T) {
	const floor = "go1.25.10"
	v := runtime.Version()
	if !strings.HasPrefix(v, "go") {
		t.Skipf("non-standard go version string %q", v)
	}
	if cmpGoVersions(v, floor) < 0 {
		t.Errorf("Go toolchain %s is older than the security floor %s. Bump the toolchain directive in backend/go.mod and rerun govulncheck.", v, floor)
	}
}

// TestNoVulnerablePinnedDeps reads backend/go.mod (via the test binary's
// build info) and fails if quic-go or golang.org/x/net are pinned at a
// version known to be exploitable.
func TestNoVulnerablePinnedDeps(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("build info unavailable")
	}
	mins := map[string]string{
		"github.com/quic-go/quic-go": "v0.57.0",
		"golang.org/x/net":           "v0.53.0",
	}
	for _, dep := range info.Deps {
		if dep == nil {
			continue
		}
		minVer, ok := mins[dep.Path]
		if !ok {
			continue
		}
		if cmpModuleVersions(dep.Version, minVer) < 0 {
			t.Errorf("%s pinned at %s; minimum safe version is %s (see govulncheck)", dep.Path, dep.Version, minVer)
		}
	}
}

// cmpGoVersions compares "go1.25.10"-style strings. Returns -1/0/+1.
func cmpGoVersions(a, b string) int {
	return cmpDotted(strings.TrimPrefix(a, "go"), strings.TrimPrefix(b, "go"))
}

// cmpModuleVersions compares "v0.57.0" / "v0.53.0"-style strings. Returns
// -1/0/+1. Pseudo-versions (with `-` suffixes) are compared lexically
// after the numeric prefix, which is a best-effort fallback.
func cmpModuleVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	if i := strings.IndexByte(a, '-'); i >= 0 {
		a = a[:i]
	}
	if i := strings.IndexByte(b, '-'); i >= 0 {
		b = b[:i]
	}
	return cmpDotted(a, b)
}

func cmpDotted(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < len(ap) || i < len(bp); i++ {
		var av, bv int
		if i < len(ap) {
			av = atoi(ap[i])
		}
		if i < len(bp) {
			bv = atoi(bp[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// Sanity check that go.mod exists where we expect — guards against the
// test being run from an unexpected working directory.
func TestGoModPresent(t *testing.T) {
	candidates := []string{
		"../../go.mod",
		"go.mod",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return
		}
	}
	t.Errorf("go.mod not found relative to test; tried %v from %s", candidates, mustGetwd(t))
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(wd)
}
