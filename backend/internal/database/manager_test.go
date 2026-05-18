package database

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildPostgresURLEncodesCredentials(t *testing.T) {
	// User and password contain every character that would otherwise
	// break URL parsing or smuggle host/query components.
	got := buildPostgresURL(
		"u@ser/name",
		"p@ss:wo/rd?#&",
		"db.example.com",
		5432,
		"app db",
		"disable",
	)

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("output is not a valid URL: %v\n%s", err, got)
	}
	if u.Host != "db.example.com:5432" {
		t.Errorf("Host=%q, want db.example.com:5432 (credentials must not redirect host)", u.Host)
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	if user != "u@ser/name" || pass != "p@ss:wo/rd?#&" {
		t.Errorf("decoded user/pass mismatch: user=%q pass=%q", user, pass)
	}
	if u.Path != "/app db" {
		t.Errorf("Path=%q, want /app db", u.Path)
	}
	if u.Query().Get("sslmode") != "disable" {
		t.Errorf("sslmode lost: %s", u.RawQuery)
	}
	// Final sanity: no raw `@` outside the userinfo component.
	rest := got[len("postgres://"):]
	at := strings.LastIndex(rest, "@")
	if at < 0 {
		t.Fatalf("no @ separator: %s", got)
	}
	if strings.Contains(rest[at+1:], "@") {
		t.Errorf("multiple @ in URL — host may be hijackable: %s", got)
	}
}

func TestBuildPostgresURLOmitsSSLModeWhenEmpty(t *testing.T) {
	got := buildPostgresURL("u", "p", "h", 5432, "d", "")
	if strings.Contains(got, "sslmode=") {
		t.Errorf("empty sslMode should not emit query param: %s", got)
	}
}
