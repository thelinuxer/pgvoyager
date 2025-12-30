package handlers

import (
	"net/http"

	"github.com/thelinuxer/pgvoyager/internal/database"
	"github.com/thelinuxer/pgvoyager/internal/models"
	"github.com/gin-gonic/gin"
)

func ListSavedQueries(c *gin.Context) {
	manager := database.GetQueryManager()
	queries := manager.List()
	c.JSON(http.StatusOK, queries)
}

func CreateSavedQuery(c *gin.Context) {
	var req models.SavedQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query, err := database.GetQueryManager().Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, query)
}

func GetSavedQuery(c *gin.Context) {
	id := c.Param("id")
	query, err := database.GetQueryManager().Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, query)
}

func UpdateSavedQuery(c *gin.Context) {
	id := c.Param("id")
	var req models.SavedQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query, err := database.GetQueryManager().Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, query)
}

func DeleteSavedQuery(c *gin.Context) {
	id := c.Param("id")
	if err := database.GetQueryManager().Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Query deleted"})
}
