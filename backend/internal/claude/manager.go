package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
	"github.com/thelinuxer/pgvoyager/internal/database"
)

// Manager handles Claude Code terminal sessions
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

var (
	manager *Manager
	once    sync.Once
)

// GetManager returns the singleton session manager
func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{
			sessions: make(map[string]*Session),
		}
	})
	return manager
}

// getBackendURL returns the backend URL based on environment variables
func getBackendURL() string {
	port := os.Getenv("PGVOYAGER_PORT")
	if port == "" {
		port = "5137"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// findMCPServer looks for the pgvoyager-mcp binary in multiple locations
func findMCPServer() string {
	// Check environment variable first
	if path := os.Getenv("PGVOYAGER_MCP_PATH"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Get current working directory
	cwd, _ := os.Getwd()

	// List of paths to check
	searchPaths := []string{
		// Relative to current working directory
		filepath.Join(cwd, "bin", "pgvoyager-mcp"),
		filepath.Join(cwd, "..", "bin", "pgvoyager-mcp"),
		filepath.Join(cwd, "pgvoyager-mcp"),
		// Relative to executable
		"",
	}

	// Add path relative to executable
	if execPath, err := os.Executable(); err == nil {
		searchPaths[len(searchPaths)-1] = filepath.Join(filepath.Dir(execPath), "pgvoyager-mcp")
	}

	// Check PATH
	if path, err := exec.LookPath("pgvoyager-mcp"); err == nil {
		return path
	}

	// Check each search path
	for _, path := range searchPaths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// MCPConfig represents the MCP configuration for Claude Code
type MCPConfig struct {
	McpServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// DatabaseContext holds information about the database for the system prompt
type DatabaseContext struct {
	Name    string
	Host    string
	Port    int
	Schemas []SchemaInfo
}

// SchemaInfo holds schema and table information
type SchemaInfo struct {
	Name   string
	Tables []TableInfo
}

// TableInfo holds basic table information
type TableInfo struct {
	Name string
}

// fetchDatabaseContext retrieves schema and table information for the system prompt
func fetchDatabaseContext(connectionID string) (*DatabaseContext, error) {
	dbManager := database.GetManager()

	conn, err := dbManager.Get(connectionID)
	if err != nil {
		return nil, err
	}

	pool, err := dbManager.GetPool(connectionID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbContext := &DatabaseContext{
		Name: conn.Database,
		Host: conn.Host,
		Port: conn.Port,
	}

	// Fetch schemas and tables (without columns to avoid argument length limits)
	query := `
		SELECT DISTINCT
			n.nspname as schema_name,
			c.relname as table_name
		FROM pg_catalog.pg_namespace n
		JOIN pg_catalog.pg_class c ON c.relnamespace = n.oid
		WHERE c.relkind = 'r'
		  AND n.nspname NOT LIKE 'pg_%'
		  AND n.nspname != 'information_schema'
		ORDER BY n.nspname, c.relname
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		// Return partial context if query fails
		return dbContext, nil
	}
	defer rows.Close()

	schemaMap := make(map[string]*SchemaInfo)
	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			continue
		}

		if _, ok := schemaMap[schemaName]; !ok {
			schemaMap[schemaName] = &SchemaInfo{Name: schemaName}
		}
		schemaMap[schemaName].Tables = append(schemaMap[schemaName].Tables, TableInfo{
			Name: tableName,
		})
	}

	for _, schema := range schemaMap {
		dbContext.Schemas = append(dbContext.Schemas, *schema)
	}

	return dbContext, nil
}

// buildSystemPrompt creates a system prompt with database context
func buildSystemPrompt(dbContext *DatabaseContext) string {
	var sb strings.Builder

	sb.WriteString("You are a PostgreSQL database assistant integrated with PgVoyager.\n\n")
	sb.WriteString(fmt.Sprintf("Initially connected to database: %s (host: %s, port: %d)\n",
		dbContext.Name, dbContext.Host, dbContext.Port))
	sb.WriteString("Note: The user may switch database connections during our conversation. Use get_connection_info to check the current connection.\n\n")

	if len(dbContext.Schemas) > 0 {
		// Count total tables
		totalTables := 0
		for _, schema := range dbContext.Schemas {
			totalTables += len(schema.Tables)
		}

		sb.WriteString("DATABASE OVERVIEW:\n")
		sb.WriteString("==================\n")
		sb.WriteString(fmt.Sprintf("Schemas: %d, Tables: %d\n\n", len(dbContext.Schemas), totalTables))

		// Only list schema names with table counts (not individual tables)
		sb.WriteString("Schemas:\n")
		for _, schema := range dbContext.Schemas {
			sb.WriteString(fmt.Sprintf("  - %s (%d tables)\n", schema.Name, len(schema.Tables)))
		}
		sb.WriteString("\nUse list_tables and get_columns tools to explore table details.\n\n")
	}

	sb.WriteString("\nYou have access to PgVoyager MCP tools:\n")
	sb.WriteString("- get_connection_info: Get info about the current database connection\n")
	sb.WriteString("- list_schemas: List all database schemas\n")
	sb.WriteString("- list_tables: List tables (optionally filter by schema)\n")
	sb.WriteString("- get_columns: Get detailed column info for a table\n")
	sb.WriteString("- get_table_info: Get table details (size, row count, etc.)\n")
	sb.WriteString("- execute_query: Run SQL queries\n")
	sb.WriteString("- list_views: List database views\n")
	sb.WriteString("- list_functions: List database functions\n")
	sb.WriteString("- get_foreign_keys: Get FK relationships\n")
	sb.WriteString("- get_indexes: Get index information\n\n")
	sb.WriteString("Editor tools (to interact with the SQL query editor):\n")
	sb.WriteString("- get_editor_content: Get the current content of the SQL editor\n")
	sb.WriteString("- insert_to_editor: Insert SQL text into the editor\n")
	sb.WriteString("- replace_editor_content: Replace the entire editor content\n\n")
	sb.WriteString("IMPORTANT: When you write SQL queries for the user, use insert_to_editor or replace_editor_content to put the query in the editor.\n")
	sb.WriteString("Use these tools to help users explore their database, write queries, and understand their data.\n")
	sb.WriteString("When writing SQL, always use fully qualified table names (schema.table) when the schema is not 'public'.\n")

	return sb.String()
}

// CreateSession spawns a new Claude Code terminal session
func (m *Manager) CreateSession(connectionID string) (*Session, error) {
	sessionID := uuid.New().String()

	// Get database connection details for system prompt
	dbManager := database.GetManager()
	conn, err := dbManager.Get(connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Find claude executable
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Find the MCP server binary
	mcpServerPath := findMCPServer()
	if mcpServerPath == "" {
		return nil, fmt.Errorf("MCP server (pgvoyager-mcp) not found. Please run 'make build' first")
	}

	// Fetch database context for system prompt
	dbContext, err := fetchDatabaseContext(connectionID)
	if err != nil {
		// Continue with minimal context
		dbContext = &DatabaseContext{
			Name: conn.Database,
			Host: conn.Host,
			Port: conn.Port,
		}
	}

	// Build system prompt
	systemPrompt := buildSystemPrompt(dbContext)

	// Create MCP configuration as JSON string
	// MCP server calls backend API using session ID - no direct DB connection
	mcpConfig := MCPConfig{
		McpServers: map[string]MCPServerConfig{
			"pgvoyager": {
				Command: mcpServerPath,
				Env: map[string]string{
					"PGVOYAGER_SESSION_ID":   sessionID,
					"PGVOYAGER_BACKEND_URL":  getBackendURL(),
				},
			},
		},
	}
	mcpConfigJSON, err := json.Marshal(mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	// Build command with arguments
	// Auto-approve all pgvoyager MCP tools
	allowedTools := []string{
		"mcp__pgvoyager__get_connection_info",
		"mcp__pgvoyager__list_schemas",
		"mcp__pgvoyager__list_tables",
		"mcp__pgvoyager__get_columns",
		"mcp__pgvoyager__get_table_info",
		"mcp__pgvoyager__execute_query",
		"mcp__pgvoyager__list_views",
		"mcp__pgvoyager__list_functions",
		"mcp__pgvoyager__get_foreign_keys",
		"mcp__pgvoyager__get_indexes",
		// Editor tools
		"mcp__pgvoyager__get_editor_content",
		"mcp__pgvoyager__insert_to_editor",
		"mcp__pgvoyager__replace_editor_content",
	}

	cmd := exec.Command(claudePath,
		"--mcp-config", string(mcpConfigJSON),
		"--append-system-prompt", systemPrompt,
		"--allowedTools", strings.Join(allowedTools, ","),
	)

	// Set environment with proper terminal settings
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PGVOYAGER_CONNECTION_ID=%s", connectionID),
		fmt.Sprintf("PGVOYAGER_SESSION_ID=%s", sessionID),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	// Start with PTY and set initial size
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: 24,
		Cols: 80,
		X:    0,
		Y:    0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	session := &Session{
		ID:           sessionID,
		ConnectionID: connectionID,
		PTY:          ptmx,
		Cmd:          cmd,
		EditorState:  &EditorState{Content: ""},
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[sessionID]
	return session, ok
}

// DestroySession terminates a session and cleans up resources
func (m *Manager) DestroySession(sessionID string) error {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	// Close PTY
	if session.PTY != nil {
		session.PTY.Close()
	}

	// Kill the process if still running
	if session.Cmd != nil && session.Cmd.Process != nil {
		session.Cmd.Process.Kill()
		session.Cmd.Wait()
	}

	// Clean up temp directory if exists
	if session.TempDir != "" {
		os.RemoveAll(session.TempDir)
	}

	return nil
}

// UpdateEditorState updates the editor state for a session
func (m *Manager) UpdateEditorState(sessionID string, state *EditorState) error {
	session, ok := m.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	session.EditorState = state
	session.mu.Unlock()

	return nil
}

// GetEditorState retrieves the current editor state for a session
func (m *Manager) GetEditorState(sessionID string) (*EditorState, error) {
	session, ok := m.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.EditorState, nil
}

// ResizePTY resizes the PTY for a session
func (m *Manager) ResizePTY(sessionID string, cols, rows int) error {
	session, ok := m.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.PTY == nil {
		return fmt.Errorf("PTY not initialized")
	}

	return pty.Setsize(session.PTY, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// UpdateSessionConnection updates the database connection for a session
// This allows MCP tools to use a different database without restarting Claude
func (m *Manager) UpdateSessionConnection(sessionID, connectionID string) error {
	session, ok := m.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Verify the new connection exists and is active
	dbManager := database.GetManager()
	if _, err := dbManager.GetPool(connectionID); err != nil {
		return fmt.Errorf("connection not available: %w", err)
	}

	session.mu.Lock()
	session.ConnectionID = connectionID
	session.mu.Unlock()

	return nil
}

// SendEditorAction sends an editor action to the frontend via WebSocket
func (m *Manager) SendEditorAction(sessionID string, action *EditorActionData) error {
	session, ok := m.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.RLock()
	conn := session.WSConn
	session.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("no WebSocket connection for session")
	}

	msg := WSMessage{
		Type: "editor_action",
		Data: action,
	}

	session.wsMu.Lock()
	defer session.wsMu.Unlock()

	return conn.WriteJSON(msg)
}
