package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
	cachedRelease     *GitHubRelease
	cacheTime         time.Time
	cacheDuration     = 5 * time.Minute
)

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
	if cachedRelease != nil && time.Since(cacheTime) < cacheDuration {
		c.JSON(http.StatusOK, buildUpdateResponse(currentVersion, cachedRelease))
		return
	}

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
	cachedRelease = release
	cacheTime = time.Now()

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
