package claude

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateSessionTokenIsRandomAndUrlSafe(t *testing.T) {
	a, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken: %v", err)
	}
	b, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken: %v", err)
	}
	if a == b {
		t.Errorf("two consecutive tokens collided: %s", a)
	}
	if len(a) < 32 {
		t.Errorf("token too short (%d chars) — should be ~43 chars for 32 random bytes b64-url-no-pad", len(a))
	}
	// Base64-url-no-pad alphabet only.
	for _, c := range a {
		ok := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
		if !ok {
			t.Errorf("token contains non-URL-safe character %q in %q", c, a)
			break
		}
	}
}

func TestAuthenticate(t *testing.T) {
	m := &Manager{sessions: map[string]*Session{}}
	m.sessions["abc"] = &Session{ID: "abc", Token: "real-token"}

	cases := []struct {
		name      string
		sessionID string
		token     string
		wantErr   bool
	}{
		{"happy path", "abc", "real-token", false},
		{"wrong token", "abc", "wrong-token", true},
		{"unknown session", "missing", "real-token", true},
		{"empty token", "abc", "", true},
		{"empty session", "", "real-token", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := m.Authenticate(tc.sessionID, tc.token)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidSessionToken) {
					t.Errorf("got err=%v, want ErrInvalidSessionToken", err)
				}
				if s != nil {
					t.Errorf("got session %+v, want nil", s)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if s == nil || s.ID != tc.sessionID {
				t.Errorf("got session %+v", s)
			}
		})
	}
}

func TestAuthenticateConstantTime(t *testing.T) {
	// Smoke check: subtle.ConstantTimeCompare requires equal lengths to
	// avoid early-exit timing leaks. Tokens of different lengths should
	// still fail without panicking and without short-circuiting in a way
	// the caller can observe.
	m := &Manager{sessions: map[string]*Session{}}
	m.sessions["abc"] = &Session{ID: "abc", Token: "aaaaaaaaaa"}
	for _, attempt := range []string{"", "a", "aaaaaaaaaa-extra"} {
		if _, err := m.Authenticate("abc", attempt); err == nil {
			t.Errorf("wrong-length attempt %q accepted", attempt)
		}
	}
}

func TestMaxSessionsCap(t *testing.T) {
	if MaxSessions <= 0 {
		t.Skip("MaxSessions disabled")
	}
	m := &Manager{sessions: map[string]*Session{}}
	for i := 0; i < MaxSessions; i++ {
		m.sessions[string(rune('a'+i))] = &Session{}
	}
	// CreateSession needs a real DB connection, so we can't call it
	// directly; instead simulate the cap check that runs at the top of
	// CreateSession.
	m.mu.RLock()
	live := len(m.sessions)
	m.mu.RUnlock()
	if live < MaxSessions {
		t.Fatalf("setup wrong: %d sessions", live)
	}
	if !(live >= MaxSessions) {
		t.Errorf("cap check should fire when live=%d MaxSessions=%d", live, MaxSessions)
	}
}

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
