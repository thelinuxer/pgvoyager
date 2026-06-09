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
		// CSP. `'unsafe-inline'` is allowed on script-src and style-src
		// because SvelteKit emits an inline bootstrap <script> (and
		// inline style attributes) at the end of index.html — without
		// it the SPA loads HTML but never hydrates. Practical risk is
		// low for a local-only tool whose entire bundle is shipped in
		// our binary (no injection surface). If we later add a CSP
		// nonce or hash-based scheme, this can be tightened.
		//
		// connect-src: `'self'` covers same-origin fetch/XHR but does
		// NOT cover ws:// in all browsers. The explicit ws://localhost:*
		// and ws://127.0.0.1:* entries are required because the desktop
		// binary binds a dynamic (OS-assigned) port, so we cannot pin a
		// specific port in the CSP. In dev mode the Vite frontend on
		// port 5173 connects to ws://localhost:5137 directly; removing
		// these entries would break the Claude terminal WebSocket.
		// Tightening to a specific port is only possible once we commit
		// to a fixed port for both modes.
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
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

// hostOnly extracts the hostname portion of a host:port string. Falls back
// to the whole string when there is no port (SplitHostPort requires a port).
func hostOnly(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return h
}

// OriginGuard rejects requests whose Origin header doesn't match the
// AllowedOrigin policy. Used at the router level so individual handlers
// don't need to re-implement the check.
//
// DNS-rebinding defense: in addition to Origin validation, the Host header
// is checked. A DNS-rebind attack can omit the Origin header (or send one
// that passes loopback checks) while using a Host like
// "pgvoyager.attacker.com:5137". If the Host resolves to the server IP, the
// request arrives — but our server must refuse it when the Host is neither
// loopback nor the configured bind address.
func OriginGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- Host header validation (DNS-rebinding defense) ---
		// Extract the bare hostname from the Host header (strip port if any).
		// Empty Host is allowed (e.g. direct TCP clients that omit the header).
		if reqHost := hostOnly(c.Request.Host); reqHost != "" &&
			!IsLoopback(reqHost) &&
			!strings.EqualFold(reqHost, ListenHost()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("host %q not allowed", reqHost),
			})
			return
		}

		// --- Origin header validation (CSRF defense) ---
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
