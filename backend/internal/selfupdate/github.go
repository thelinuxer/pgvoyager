package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/thelinuxer/pgvoyager/internal/version"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

var latestReleaseURL = "https://api.github.com/repos/" + version.GitHubRepo + "/releases/latest"

var tagPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+([.\-+][0-9A-Za-z.\-]+)?$`)

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
	if !tagPattern.MatchString(rel.TagName) {
		return "", "", fmt.Errorf("selfupdate: release tag %q does not match expected pattern", rel.TagName)
	}
	return rel.TagName, rel.HTMLURL, nil
}

// compareVersions compares dotted numeric versions: -1 if a<b, 0 equal, 1 a>b.
func compareVersions(a, b string) int {
	a = strings.SplitN(a, "-", 2)[0]
	b = strings.SplitN(b, "-", 2)[0]
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
