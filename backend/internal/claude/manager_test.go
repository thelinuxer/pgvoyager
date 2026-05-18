package claude

import (
	"strings"
	"testing"
)

func TestBuildSubprocessEnvDropsSecrets(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/home/u",
		"TERM=xterm-256color",
		"LC_ALL=en_US.UTF-8",
		"ANTHROPIC_API_KEY=anthropic-key",
		"AWS_REGION=us-east-1",
		// Secrets that must NOT propagate:
		"PGPASSWORD=hunter2",
		"AWS_SECRET_ACCESS_KEY=aws-secret",
		"AWS_SESSION_TOKEN=aws-session",
		"GITHUB_TOKEN=ghp_x",
		"SLACK_TOKEN=xoxb-x",
		"DATABASE_URL=postgres://u:p@h/d",
		"OPENAI_API_KEY=sk-x",
	}
	got := buildSubprocessEnv(parent,
		"PGVOYAGER_SESSION_ID=abc",
	)

	mustHave := []string{
		"PATH=/usr/bin",
		"HOME=/home/u",
		"TERM=xterm-256color",
		"LC_ALL=en_US.UTF-8",
		"ANTHROPIC_API_KEY=anthropic-key",
		"AWS_REGION=us-east-1",
		"PGVOYAGER_SESSION_ID=abc",
	}
	for _, kv := range mustHave {
		if !contains(got, kv) {
			t.Errorf("buildSubprocessEnv dropped required var %q\ngot: %v", kv, got)
		}
	}

	mustDrop := []string{
		"PGPASSWORD",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"GITHUB_TOKEN",
		"SLACK_TOKEN",
		"DATABASE_URL",
		"OPENAI_API_KEY",
	}
	for _, name := range mustDrop {
		for _, kv := range got {
			if strings.HasPrefix(kv, name+"=") {
				t.Errorf("buildSubprocessEnv leaked secret %q to subprocess", name)
			}
		}
	}
}

func TestBuildSubprocessEnvIgnoresMalformed(t *testing.T) {
	got := buildSubprocessEnv([]string{
		"=novarname",
		"NOEQUAL",
		"PATH=/bin",
	})
	for _, kv := range got {
		if kv == "=novarname" || kv == "NOEQUAL" {
			t.Errorf("buildSubprocessEnv kept malformed entry %q", kv)
		}
	}
	if !contains(got, "PATH=/bin") {
		t.Errorf("buildSubprocessEnv dropped PATH")
	}
}

func TestBuildSubprocessEnvAdditionsOverride(t *testing.T) {
	// Additions are appended last; exec.Cmd uses the last occurrence, so
	// callers can override even an allowlisted value.
	got := buildSubprocessEnv(
		[]string{"TERM=dumb"},
		"TERM=xterm-256color",
	)
	last := ""
	for _, kv := range got {
		if strings.HasPrefix(kv, "TERM=") {
			last = kv
		}
	}
	if last != "TERM=xterm-256color" {
		t.Errorf("TERM override not preserved as last entry: %v", got)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
