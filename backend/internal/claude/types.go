package claude

import (
	"os"
	"os/exec"
	"sync"

	"github.com/gorilla/websocket"
)

// Session represents a Claude Code terminal session
type Session struct {
	ID           string
	ConnectionID string   // Active database connection ID
	PTY          *os.File // PTY master file descriptor
	Cmd          *exec.Cmd
	EditorState  *EditorState
	TempDir      string // Temporary directory for MCP config
	WSConn       *websocket.Conn // WebSocket connection to frontend
	mu           sync.RWMutex
	wsMu         sync.Mutex // Mutex for WebSocket writes
}

// EditorState holds the current state of the SQL editor
type EditorState struct {
	Content   string     `json:"content"`
	Selection *Selection `json:"selection,omitempty"`
	Cursor    *Position  `json:"cursor,omitempty"`
}

// Selection represents a text selection range
type Selection struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Position represents a cursor position
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// WebSocket message types
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Terminal input data
type InputData struct {
	Data string `json:"data"`
}

// Terminal resize data
type ResizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// EditorUpdateData for syncing editor state
type EditorUpdateData struct {
	Content   string     `json:"content"`
	Selection *Selection `json:"selection,omitempty"`
	Cursor    *Position  `json:"cursor,omitempty"`
}

// EditorActionData for actions from Claude to editor
type EditorActionData struct {
	Action   string `json:"action"` // "insert", "replace"
	Text     string `json:"text"`
	Position *Position `json:"position,omitempty"`
}

// CreateSessionRequest for creating a new session
type CreateSessionRequest struct {
	ConnectionID string `json:"connectionId" binding:"required"`
}

// CreateSessionResponse returned after session creation
type CreateSessionResponse struct {
	SessionID string `json:"sessionId"`
}
