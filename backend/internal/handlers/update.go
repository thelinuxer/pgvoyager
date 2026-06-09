package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thelinuxer/pgvoyager/internal/selfupdate"
	"github.com/thelinuxer/pgvoyager/internal/version"
)

// GitHubRelease represents the GitHub API response for a release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

// UpdateCheckResponse is the response for update check endpoint
type UpdateCheckResponse struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseURL     string `json:"releaseUrl"`
}

// cache for rate limiting GitHub API calls
var (
	cachedRelease  *GitHubRelease
	cacheTime      time.Time
	cacheDuration  = 5 * time.Minute
	cacheMu        sync.Mutex
)

// restartTokenOnce and restartTokenVal hold a per-process random token used to
// guard the /api/update/restart endpoint against CSRF. A malicious local page
// can POST cross-origin (OriginGuard allows loopback) but cannot READ the
// /api/update/status response cross-origin (CORS only exposes dev origins),
// so it cannot learn this token.
var (
	restartTokenOnce sync.Once
	restartTokenVal  string
)

func restartToken() string {
	restartTokenOnce.Do(func() {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("pgvoyager: failed to generate restart token: " + err.Error())
		}
		restartTokenVal = base64.RawURLEncoding.EncodeToString(b)
	})
	return restartTokenVal
}

// GetVersion returns the current version
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": version.Version,
	})
}

// CheckUpdate checks for available updates
func CheckUpdate(c *gin.Context) {
	currentVersion := version.Version

	// Use cached result if available and fresh
	cacheMu.Lock()
	if cachedRelease != nil && time.Since(cacheTime) < cacheDuration {
		resp := buildUpdateResponse(currentVersion, cachedRelease)
		cacheMu.Unlock()
		c.JSON(http.StatusOK, resp)
		return
	}
	cacheMu.Unlock()

	// Fetch latest release from GitHub
	release, err := fetchLatestRelease()
	if err != nil {
		c.JSON(http.StatusOK, UpdateCheckResponse{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
			ReleaseURL:     version.ReleasesURL(),
		})
		return
	}

	// Cache the result
	cacheMu.Lock()
	cachedRelease = release
	cacheTime = time.Now()
	cacheMu.Unlock()

	c.JSON(http.StatusOK, buildUpdateResponse(currentVersion, release))
}

func fetchLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", version.LatestReleaseAPIURL(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "PgVoyager/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func buildUpdateResponse(currentVersion string, release *GitHubRelease) UpdateCheckResponse {
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	hasUpdate := false
	if currentClean != "dev" && latestVersion != currentClean {
		hasUpdate = compareVersions(currentClean, latestVersion) < 0
	}

	return UpdateCheckResponse{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		ReleaseURL:     release.HTMLURL,
	}
}

// compareVersions compares two semantic versions
// Returns -1 if v1 < v2, 0 if equal, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		var n1, n2 int
		fmt.Sscanf(parts1[i], "%d", &n1)
		fmt.Sscanf(parts2[i], "%d", &n2)

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	if len(parts1) < len(parts2) {
		return -1
	}
	if len(parts1) > len(parts2) {
		return 1
	}

	return 0
}

// updateManager is set by the desktop binary; nil for the server edition.
var updateManager *selfupdate.Manager

// SetUpdateManager wires the desktop self-update manager into the handlers.
func SetUpdateManager(m *selfupdate.Manager) { updateManager = m }

// UpdateStatus returns the current self-update state. Desktop edition reports
// the live manager state; server edition reports a computed check result.
func UpdateStatus(c *gin.Context) {
	if updateManager != nil {
		st := updateManager.Status()
		c.JSON(http.StatusOK, gin.H{
			"edition":        st.Edition,
			"status":         st.Status,
			"currentVersion": st.CurrentVersion,
			"latestVersion":  st.LatestVersion,
			"releaseUrl":     st.ReleaseURL,
			"needsElevation": st.NeedsElevation,
			"error":          st.Error,
			"restartToken":   restartToken(),
		})
		return
	}
	c.JSON(http.StatusOK, computeServerStatus())
}

func computeServerStatus() gin.H {
	current := version.Version
	rel, err := fetchLatestRelease()
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

// UpdateRestart applies a staged update (desktop edition only). It responds
// first, then applies in a goroutine after a short delay so the HTTP response
// flushes before the process swaps itself and tears down.
func UpdateRestart(c *gin.Context) {
	if updateManager == nil || !version.IsDesktop() {
		c.JSON(http.StatusConflict, gin.H{"error": "self-update not supported for this build"})
		return
	}
	if !updateManager.CanRestart() {
		c.JSON(http.StatusConflict, gin.H{"error": "no staged update to apply"})
		return
	}
	// Require the per-process CSRF token that the frontend reads from UpdateStatus.
	// A malicious local page can POST cross-origin but cannot read the status
	// response cross-origin (CORS restricts reads to dev origins), so it cannot
	// forge this header.
	provided := c.GetHeader("X-Update-Token")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(restartToken())) != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid restart token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"restarting": true})
	go func() {
		time.Sleep(300 * time.Millisecond) // let the response flush before teardown
		if err := updateManager.Restart(); err != nil {
			log.Printf("self-update restart failed: %v", err)
		}
	}()
}
