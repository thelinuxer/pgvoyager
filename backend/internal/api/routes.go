package api

import (
	"github.com/atoulan/pgvoyager/internal/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Connection management
		connections := api.Group("/connections")
		{
			connections.GET("", handlers.ListConnections)
			connections.POST("", handlers.CreateConnection)
			connections.POST("/test", handlers.TestConnection)
			connections.GET("/:id", handlers.GetConnection)
			connections.PUT("/:id", handlers.UpdateConnection)
			connections.DELETE("/:id", handlers.DeleteConnection)
			connections.POST("/:id/connect", handlers.Connect)
			connections.POST("/:id/disconnect", handlers.Disconnect)
		}

		// Schema browsing (requires active connection)
		schema := api.Group("/schema/:connId")
		{
			schema.GET("/databases", handlers.ListDatabases)
			schema.GET("/schemas", handlers.ListSchemas)
			schema.GET("/tables", handlers.ListTables)
			schema.GET("/tables/:schema/:table", handlers.GetTableInfo)
			schema.GET("/tables/:schema/:table/columns", handlers.GetTableColumns)
			schema.GET("/tables/:schema/:table/constraints", handlers.GetTableConstraints)
			schema.GET("/tables/:schema/:table/indexes", handlers.GetTableIndexes)
			schema.GET("/tables/:schema/:table/foreign-keys", handlers.GetForeignKeys)
			schema.GET("/views", handlers.ListViews)
			schema.GET("/functions", handlers.ListFunctions)
			schema.GET("/sequences", handlers.ListSequences)
			schema.GET("/types", handlers.ListTypes)
		}

		// Data operations
		data := api.Group("/data/:connId")
		{
			data.GET("/tables/:schema/:table", handlers.GetTableData)
			data.GET("/tables/:schema/:table/count", handlers.GetTableRowCount)
			data.GET("/fk-preview/:schema/:table/:column/:value", handlers.GetForeignKeyPreview)
		}

		// Query execution
		query := api.Group("/query/:connId")
		{
			query.POST("/execute", handlers.ExecuteQuery)
			query.POST("/explain", handlers.ExplainQuery)
		}
	}
}
