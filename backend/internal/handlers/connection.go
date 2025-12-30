package handlers

import (
	"net/http"

	"github.com/thelinuxer/pgvoyager/internal/database"
	"github.com/thelinuxer/pgvoyager/internal/models"
	"github.com/gin-gonic/gin"
)

func ListConnections(c *gin.Context) {
	manager := database.GetManager()
	connections := manager.List()
	c.JSON(http.StatusOK, connections)
}

func CreateConnection(c *gin.Context) {
	var req models.ConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conn, err := database.GetManager().Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, conn)
}

func TestConnection(c *gin.Context) {
	var req models.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SSLMode == "" {
		req.SSLMode = "prefer"
	}

	if err := database.GetManager().TestConnection(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Connection successful"})
}

func GetConnection(c *gin.Context) {
	id := c.Param("id")
	conn, err := database.GetManager().Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, conn)
}

func UpdateConnection(c *gin.Context) {
	id := c.Param("id")
	var req models.ConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conn, err := database.GetManager().Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conn)
}

func DeleteConnection(c *gin.Context) {
	id := c.Param("id")
	if err := database.GetManager().Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Connection deleted"})
}

func Connect(c *gin.Context) {
	id := c.Param("id")
	if err := database.GetManager().Connect(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Connected successfully"})
}

func Disconnect(c *gin.Context) {
	id := c.Param("id")
	if err := database.GetManager().Disconnect(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Disconnected successfully"})
}
