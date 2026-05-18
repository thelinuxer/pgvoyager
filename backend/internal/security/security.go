// Package security centralizes HTTP-server hardening: origin allowlist,
// request-size cap, security response headers, and shared helpers used by
// both the HTTP API and the Claude WebSocket upgrader.
package security

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxRequestBodyBytes caps every request body. Query-execute and DDL endpoints
// take JSON bodies; 10 MiB is generous for any realistic SQL payload and
// blocks unbounded-body DoS against ShouldBindJSON callers.
const MaxRequestBodyBytes = 10 * 1024 * 1024

// devOrigins are the localhost origins allowed when CORS/dev mode is active.
// Kept in one place so both the CORS middleware and the WebSocket Origin
// check use the same list.
var devOrigins = []string{
	"http://localhost:5137",
	"http://localhost:5173",
	"http://localhost:3000",
	"http://127.0.0.1:5137",
	"http://127.0.0.1:5173",
	"http://127.0.0.1:3000",
}

// DevOrigins returns a copy of the dev-mode origin allowlist.
func DevOrigins() []string {
	out := make([]string, len(devOrigins))
	copy(out, devOrigins)
	return out
}

// ListenHost returns the bind address: 127.0.0.1 by default, overridable via
// the PGVOYAGER_HOST env var. PgVoyager is a local dev tool — binding all
// interfaces (the prior default) exposed the full DB-admin API to the LAN.
func ListenHost() string {
	if h := strings.TrimSpace(os.Getenv("PGVOYAGER_HOST")); h != "" {
		return h
	}
	return "127.0.0.1"
}

// IsLoopback reports whether host (without port) is a loopback address.
// Used as a fallback when matching Origin headers — same-host loopback
// connections are always allowed.
func IsLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// AllowedOrigin reports whether the given Origin header value should be
// accepted for cross-origin requests / WebSocket upgrades.
//
//   - Same-host (Origin host == Request Host) is always allowed.
//   - Loopback Origin host (e.g. http://127.0.0.1:5173) is always allowed.
//   - Anything else is rejected.
//
// Empty Origin returns true (non-browser clients like curl / native MCP
// process don't send Origin).
func AllowedOrigin(origin, requestHost string) bool {
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, requestHost) {
		return true
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}
	return IsLoopback(host)
}

// SecurityHeaders is a gin middleware that sets defensive response headers
// on every response. Tight CSP because the SvelteKit bundle is self-hosted
// and never loads remote scripts/styles.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		// CSP allows only same-origin and inline styles (SvelteKit's
		// hydration injects inline style attrs); no remote fetch.
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"font-src 'self' data:; "+
				"connect-src 'self' ws://localhost:* ws://127.0.0.1:*; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")
		c.Next()
	}
}

// MaxBodyBytes is a gin middleware that caps request body size. Skips the
// Claude terminal WebSocket route, which streams indefinitely.
func MaxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// WebSocket upgrades read raw frames after the upgrade; the
		// HTTP body cap doesn't apply once the connection switches
		// protocols. Skip the route explicitly.
		if strings.HasPrefix(c.Request.URL.Path, "/api/claude/terminal/") {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

// OriginGuard rejects requests whose Origin header doesn't match the
// AllowedOrigin policy. Used at the router level so individual handlers
// don't need to re-implement the check.
func OriginGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if !AllowedOrigin(origin, c.Request.Host) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("origin %q not allowed", origin),
			})
			return
		}
		c.Next()
	}
}
