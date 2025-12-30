package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	backendURL string
	sessionID  string
)

func main() {
	// Get backend URL and session ID from environment
	backendURL = os.Getenv("PGVOYAGER_BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8081"
	}

	sessionID = os.Getenv("PGVOYAGER_SESSION_ID")
	if sessionID == "" {
		fmt.Fprintln(os.Stderr, "PGVOYAGER_SESSION_ID environment variable not set")
		os.Exit(1)
	}

	// Create MCP server
	s := server.NewMCPServer(
		"PgVoyager Database Tools",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Register tools
	registerDatabaseTools(s)

	// Start server using stdio transport
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// callBackendAPI makes a request to the PgVoyager backend
func callBackendAPI(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = strings.NewReader(string(jsonBody))
	}

	url := fmt.Sprintf("%s%s", backendURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Claude-Session-ID", sessionID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func registerDatabaseTools(s *server.MCPServer) {
	// Editor tools
	getEditorContent := mcp.NewTool("get_editor_content",
		mcp.WithDescription("Get the current content of the SQL query editor. Use this to see what query the user is working on."),
	)
	s.AddTool(getEditorContent, handleGetEditorContent)

	insertToEditor := mcp.NewTool("insert_to_editor",
		mcp.WithDescription("Insert text into the SQL query editor. Use this to add SQL queries or code snippets for the user."),
		mcp.WithString("text", mcp.Required(), mcp.Description("The text to insert into the editor")),
		mcp.WithNumber("line", mcp.Description("Optional line number to insert at (0-based). If not specified, appends to end.")),
		mcp.WithNumber("column", mcp.Description("Optional column number to insert at (0-based)")),
	)
	s.AddTool(insertToEditor, handleInsertToEditor)

	replaceEditorContent := mcp.NewTool("replace_editor_content",
		mcp.WithDescription("Replace the entire content of the SQL query editor. Use this when you want to provide a complete new query."),
		mcp.WithString("content", mcp.Required(), mcp.Description("The new content for the editor")),
	)
	s.AddTool(replaceEditorContent, handleReplaceEditorContent)

	// List schemas tool
	listSchemas := mcp.NewTool("list_schemas",
		mcp.WithDescription("List all database schemas in the currently connected database. Returns schema names, owners, and table counts."),
	)
	s.AddTool(listSchemas, handleListSchemas)

	// List tables tool
	listTables := mcp.NewTool("list_tables",
		mcp.WithDescription("List tables in the currently connected database. Optionally filter by schema."),
		mcp.WithString("schema", mcp.Description("Optional schema name to filter tables")),
	)
	s.AddTool(listTables, handleListTables)

	// Get columns tool
	getColumns := mcp.NewTool("get_columns",
		mcp.WithDescription("Get column information for a specific table, including data types, constraints, and foreign key references."),
		mcp.WithString("schema", mcp.Required(), mcp.Description("The schema containing the table")),
		mcp.WithString("table", mcp.Required(), mcp.Description("The table name")),
	)
	s.AddTool(getColumns, handleGetColumns)

	// Get table info tool
	getTableInfo := mcp.NewTool("get_table_info",
		mcp.WithDescription("Get detailed information about a table including row count, size, and constraints."),
		mcp.WithString("schema", mcp.Required(), mcp.Description("The schema containing the table")),
		mcp.WithString("table", mcp.Required(), mcp.Description("The table name")),
	)
	s.AddTool(getTableInfo, handleGetTableInfo)

	// Execute query tool
	executeQuery := mcp.NewTool("execute_query",
		mcp.WithDescription("Execute a SQL query on the currently connected database and return the results. Use this to run SELECT queries to explore data. Be careful with INSERT/UPDATE/DELETE queries."),
		mcp.WithString("sql", mcp.Required(), mcp.Description("The SQL query to execute")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of rows to return (default 100, max 1000)")),
	)
	s.AddTool(executeQuery, handleExecuteQuery)

	// List views tool
	listViews := mcp.NewTool("list_views",
		mcp.WithDescription("List database views in the currently connected database. Optionally filter by schema."),
		mcp.WithString("schema", mcp.Description("Optional schema name to filter views")),
	)
	s.AddTool(listViews, handleListViews)

	// List functions tool
	listFunctions := mcp.NewTool("list_functions",
		mcp.WithDescription("List database functions/procedures in the currently connected database. Optionally filter by schema."),
		mcp.WithString("schema", mcp.Description("Optional schema name to filter functions")),
	)
	s.AddTool(listFunctions, handleListFunctions)

	// Get foreign keys tool
	getForeignKeys := mcp.NewTool("get_foreign_keys",
		mcp.WithDescription("Get foreign key relationships for a table."),
		mcp.WithString("schema", mcp.Required(), mcp.Description("The schema containing the table")),
		mcp.WithString("table", mcp.Required(), mcp.Description("The table name")),
	)
	s.AddTool(getForeignKeys, handleGetForeignKeys)

	// Get indexes tool
	getIndexes := mcp.NewTool("get_indexes",
		mcp.WithDescription("Get index information for a table."),
		mcp.WithString("schema", mcp.Required(), mcp.Description("The schema containing the table")),
		mcp.WithString("table", mcp.Required(), mcp.Description("The table name")),
	)
	s.AddTool(getIndexes, handleGetIndexes)

	// Get current connection info
	getConnectionInfo := mcp.NewTool("get_connection_info",
		mcp.WithDescription("Get information about the currently active database connection."),
	)
	s.AddTool(getConnectionInfo, handleGetConnectionInfo)
}

func handleListSchemas(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := callBackendAPI(ctx, "GET", "/api/mcp/schemas", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list schemas: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schemaFilter := request.GetString("schema", "")

	endpoint := "/api/mcp/tables"
	if schemaFilter != "" {
		endpoint = fmt.Sprintf("%s?schema=%s", endpoint, schemaFilter)
	}

	resp, err := callBackendAPI(ctx, "GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list tables: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleGetColumns(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schema, err := request.RequireString("schema")
	if err != nil {
		return mcp.NewToolResultError("schema parameter is required"), nil
	}
	table, err := request.RequireString("table")
	if err != nil {
		return mcp.NewToolResultError("table parameter is required"), nil
	}

	endpoint := fmt.Sprintf("/api/mcp/tables/%s/%s/columns", schema, table)
	resp, err := callBackendAPI(ctx, "GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get columns: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleGetTableInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schema, err := request.RequireString("schema")
	if err != nil {
		return mcp.NewToolResultError("schema parameter is required"), nil
	}
	table, err := request.RequireString("table")
	if err != nil {
		return mcp.NewToolResultError("table parameter is required"), nil
	}

	endpoint := fmt.Sprintf("/api/mcp/tables/%s/%s", schema, table)
	resp, err := callBackendAPI(ctx, "GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get table info: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleExecuteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError("sql parameter is required"), nil
	}

	limit := 100
	args := request.GetArguments()
	if limitVal, ok := args["limit"].(float64); ok {
		limit = int(limitVal)
		if limit > 1000 {
			limit = 1000
		}
		if limit < 1 {
			limit = 1
		}
	}

	body := map[string]interface{}{
		"sql":   sql,
		"limit": limit,
	}

	resp, err := callBackendAPI(ctx, "POST", "/api/mcp/query", body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleListViews(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schemaFilter := request.GetString("schema", "")

	endpoint := "/api/mcp/views"
	if schemaFilter != "" {
		endpoint = fmt.Sprintf("%s?schema=%s", endpoint, schemaFilter)
	}

	resp, err := callBackendAPI(ctx, "GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list views: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleListFunctions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schemaFilter := request.GetString("schema", "")

	endpoint := "/api/mcp/functions"
	if schemaFilter != "" {
		endpoint = fmt.Sprintf("%s?schema=%s", endpoint, schemaFilter)
	}

	resp, err := callBackendAPI(ctx, "GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list functions: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleGetForeignKeys(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schema, err := request.RequireString("schema")
	if err != nil {
		return mcp.NewToolResultError("schema parameter is required"), nil
	}
	table, err := request.RequireString("table")
	if err != nil {
		return mcp.NewToolResultError("table parameter is required"), nil
	}

	endpoint := fmt.Sprintf("/api/mcp/tables/%s/%s/foreign-keys", schema, table)
	resp, err := callBackendAPI(ctx, "GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get foreign keys: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleGetIndexes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schema, err := request.RequireString("schema")
	if err != nil {
		return mcp.NewToolResultError("schema parameter is required"), nil
	}
	table, err := request.RequireString("table")
	if err != nil {
		return mcp.NewToolResultError("table parameter is required"), nil
	}

	endpoint := fmt.Sprintf("/api/mcp/tables/%s/%s/indexes", schema, table)
	resp, err := callBackendAPI(ctx, "GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get indexes: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleGetConnectionInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := callBackendAPI(ctx, "GET", "/api/mcp/connection", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get connection info: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleGetEditorContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := callBackendAPI(ctx, "GET", "/api/mcp/editor", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get editor content: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleInsertToEditor(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("text parameter is required"), nil
	}

	body := map[string]interface{}{
		"text": text,
	}

	// Optional position
	args := request.GetArguments()
	if line, ok := args["line"].(float64); ok {
		position := map[string]int{"line": int(line)}
		if column, ok := args["column"].(float64); ok {
			position["column"] = int(column)
		}
		body["position"] = position
	}

	resp, err := callBackendAPI(ctx, "POST", "/api/mcp/editor/insert", body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to insert to editor: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}

func handleReplaceEditorContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := request.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content parameter is required"), nil
	}

	body := map[string]interface{}{
		"content": content,
	}

	resp, err := callBackendAPI(ctx, "POST", "/api/mcp/editor/replace", body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to replace editor content: %v", err)), nil
	}
	return mcp.NewToolResultText(string(resp)), nil
}
