package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/thelinuxer/pgvoyager/internal/storage"
)

type AddQueryHistoryRequest struct {
	ConnectionID   string `json:"connectionId" binding:"required"`
	ConnectionName string `json:"connectionName" binding:"required"`
	SQL            string `json:"sql" binding:"required"`
	Duration       int64  `json:"duration" binding:"required"`
	RowCount       int    `json:"rowCount" binding:"required"`
	Success        bool   `json:"success"`
	Error          string `json:"error"`
}

// GetQueryHistory retrieves query history
func GetQueryHistory(c *gin.Context) {
	connectionID := c.Query("connectionId")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	entries, err := storage.GetQueryHistory(connectionID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// AddQueryHistory adds a new query to history
func AddQueryHistory(c *gin.Context) {
	var req AddQueryHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := &storage.QueryHistoryEntry{
		ID:             uuid.New().String(),
		ConnectionID:   req.ConnectionID,
		ConnectionName: req.ConnectionName,
		SQL:            req.SQL,
		Duration:       req.Duration,
		RowCount:       req.RowCount,
		Success:        req.Success,
		Error:          req.Error,
		ExecutedAt:     time.Now(),
	}

	if err := storage.AddQueryHistory(entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// DeleteQueryHistory removes a query from history
func DeleteQueryHistory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	if err := storage.DeleteQueryHistory(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ClearQueryHistory removes all query history or for a specific connection
func ClearQueryHistory(c *gin.Context) {
	connectionID := c.Query("connectionId")

	if err := storage.ClearQueryHistory(connectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cleared"})
}
