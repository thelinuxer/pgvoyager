package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thelinuxer/pgvoyager/internal/claude"
)

// authenticateSession validates a session ID + bearer token from the
// request and returns the live session. Used by every endpoint that
// mutates session-scoped state.
func authenticateSession(c *gin.Context, sessionID string) (*claude.Session, bool) {
	auth := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed Authorization header"})
		return nil, false
	}
	token := strings.TrimSpace(auth[len(prefix):])
	session, err := claude.GetManager().Authenticate(sessionID, token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session token"})
		return nil, false
	}
	return session, true
}

// CreateClaudeSession creates a new Claude Code terminal session. Returns
// both the public session ID and the per-session bearer token; the client
// must supply the token on every subsequent session-scoped request.
func CreateClaudeSession(c *gin.Context) {
	var req claude.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := claude.GetManager().CreateSession(req.ConnectionID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, claude.ErrTooManySessions) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, claude.CreateSessionResponse{
		SessionID: session.ID,
		Token:     session.Token,
	})
}

// DestroyClaudeSession terminates a Claude Code terminal session. Requires
// the per-session bearer token to prevent unauthenticated session
// destruction.
func DestroyClaudeSession(c *gin.Context) {
	sessionID := c.Param("id")
	if _, ok := authenticateSession(c, sessionID); !ok {
		return
	}

	if err := claude.GetManager().DestroySession(sessionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DestroyClaudeSessionPost is a POST version for sendBeacon compatibility on
// page close. Same auth requirements as DELETE.
func DestroyClaudeSessionPost(c *gin.Context) {
	sessionID := c.Param("id")
	if _, ok := authenticateSession(c, sessionID); !ok {
		return
	}

	// Errors here aren't actionable (we're tearing down on page close).
	_ = claude.GetManager().DestroySession(sessionID)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ClaudeTerminalWebSocket handles the WebSocket connection for terminal I/O.
// Auth is performed inside the handler (token from query param).
func ClaudeTerminalWebSocket(c *gin.Context) {
	claude.HandleTerminalWebSocket(c)
}

// UpdateClaudeSessionConnection updates the database connection for an
// existing session. Requires the per-session bearer token.
func UpdateClaudeSessionConnection(c *gin.Context) {
	sessionID := c.Param("id")
	if _, ok := authenticateSession(c, sessionID); !ok {
		return
	}

	var req struct {
		ConnectionID string `json:"connectionId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := claude.GetManager().UpdateSessionConnection(sessionID, req.ConnectionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
