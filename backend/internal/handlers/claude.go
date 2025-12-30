package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thelinuxer/pgvoyager/internal/claude"
)

// CreateClaudeSession creates a new Claude Code terminal session
func CreateClaudeSession(c *gin.Context) {
	var req claude.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := claude.GetManager().CreateSession(req.ConnectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, claude.CreateSessionResponse{
		SessionID: session.ID,
	})
}

// DestroyClaudeSession terminates a Claude Code terminal session
func DestroyClaudeSession(c *gin.Context) {
	sessionID := c.Param("id")

	if err := claude.GetManager().DestroySession(sessionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ClaudeTerminalWebSocket handles the WebSocket connection for terminal I/O
func ClaudeTerminalWebSocket(c *gin.Context) {
	claude.HandleTerminalWebSocket(c)
}

// UpdateClaudeSessionConnection updates the database connection for an existing session
func UpdateClaudeSessionConnection(c *gin.Context) {
	sessionID := c.Param("id")

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
