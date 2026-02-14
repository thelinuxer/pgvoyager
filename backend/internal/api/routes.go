package api

import (
	"github.com/thelinuxer/pgvoyager/internal/handlers"
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
			schema.GET("/all-columns", handlers.GetAllColumns)
			schema.GET("/tables/:schema/:table/constraints", handlers.GetTableConstraints)
			schema.GET("/tables/:schema/:table/indexes", handlers.GetTableIndexes)
			schema.GET("/tables/:schema/:table/foreign-keys", handlers.GetForeignKeys)
			schema.GET("/schemas/:schema/relationships", handlers.GetSchemaRelationships)
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
			// CRUD operations
			data.POST("/tables/:schema/:table/rows", handlers.InsertRow)
			data.PUT("/tables/:schema/:table/rows", handlers.UpdateRow)
			data.DELETE("/tables/:schema/:table/rows", handlers.DeleteRow)
			// Table operations
			data.DELETE("/tables/:schema/:table", handlers.DropTable)
			// Schema DDL operations
			data.POST("/schemas", handlers.CreateSchema)
			data.DELETE("/schemas/:schema", handlers.DropSchema)
			data.POST("/tables/:schema", handlers.CreateTable)
			data.POST("/tables/:schema/:table/constraints", handlers.AddConstraint)
		}

		// Query execution
		query := api.Group("/query/:connId")
		{
			query.POST("/execute", handlers.ExecuteQuery)
			query.POST("/explain", handlers.ExplainQuery)
		}

		// Database analysis
		api.GET("/analysis/:connId", handlers.RunAnalysis)

		// Query history
		history := api.Group("/history")
		{
			history.GET("", handlers.GetQueryHistory)
			history.POST("", handlers.AddQueryHistory)
			history.DELETE("/:id", handlers.DeleteQueryHistory)
			history.DELETE("", handlers.ClearQueryHistory)
		}

		// Preferences
		prefs := api.Group("/preferences")
		{
			prefs.GET("", handlers.GetPreferences)
			prefs.GET("/:key", handlers.GetPreference)
			prefs.POST("", handlers.SetPreference)
			prefs.DELETE("/:key", handlers.DeletePreference)
		}

		// Saved queries
		queries := api.Group("/queries")
		{
			queries.GET("", handlers.ListSavedQueries)
			queries.POST("", handlers.CreateSavedQuery)
			queries.GET("/:id", handlers.GetSavedQuery)
			queries.PUT("/:id", handlers.UpdateSavedQuery)
			queries.DELETE("/:id", handlers.DeleteSavedQuery)
		}

		// Claude Code terminal
		claude := api.Group("/claude")
		{
			claude.POST("/sessions", handlers.CreateClaudeSession)
			claude.DELETE("/sessions/:id", handlers.DestroyClaudeSession)
			claude.POST("/sessions/:id/destroy", handlers.DestroyClaudeSessionPost) // For sendBeacon on page close
			claude.GET("/terminal/:id", handlers.ClaudeTerminalWebSocket)
			claude.PUT("/sessions/:id/connection", handlers.UpdateClaudeSessionConnection)
		}

		// Version and updates
		api.GET("/version", handlers.GetVersion)
		api.GET("/update/check", handlers.CheckUpdate)

		// MCP API (called by MCP server, uses X-Claude-Session-ID header)
		mcp := api.Group("/mcp")
		{
			mcp.GET("/connection", handlers.MCPGetConnectionInfo)
			mcp.GET("/schemas", handlers.MCPListSchemas)
			mcp.GET("/tables", handlers.MCPListTables)
			mcp.GET("/tables/:schema/:table", handlers.MCPGetTableInfo)
			mcp.GET("/tables/:schema/:table/columns", handlers.MCPGetColumns)
			mcp.GET("/tables/:schema/:table/foreign-keys", handlers.MCPGetForeignKeys)
			mcp.GET("/tables/:schema/:table/indexes", handlers.MCPGetIndexes)
			mcp.POST("/query", handlers.MCPExecuteQuery)
			mcp.GET("/views", handlers.MCPListViews)
			mcp.GET("/functions", handlers.MCPListFunctions)
			// Editor integration
			mcp.GET("/editor", handlers.MCPGetEditorContent)
			mcp.POST("/editor/insert", handlers.MCPInsertToEditor)
			mcp.POST("/editor/replace", handlers.MCPReplaceEditorContent)
		}
	}
}
