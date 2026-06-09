package version

// Version is set at build time via ldflags
var Version = "dev"

// GitHubRepo is the repository URL
const GitHubRepo = "thelinuxer/pgvoyager"

// ReleasesURL returns the GitHub releases page URL
func ReleasesURL() string {
	return "https://github.com/" + GitHubRepo + "/releases"
}

// ReleaseTagURL returns the URL for a specific release tag
func ReleaseTagURL(tag string) string {
	return "https://github.com/" + GitHubRepo + "/releases/tag/" + tag
}

// LatestReleaseAPIURL returns the GitHub API URL for latest release
func LatestReleaseAPIURL() string {
	return "https://api.github.com/repos/" + GitHubRepo + "/releases/latest"
}

// Edition is set at build time via ldflags ("desktop" for the desktop
// wrapper, empty otherwise). It gates self-update behavior.
var Edition = ""

// IsDesktop reports whether this build is the desktop edition.
func IsDesktop() bool {
	return Edition == "desktop"
}
