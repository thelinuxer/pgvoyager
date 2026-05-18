package claude

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/thelinuxer/pgvoyager/internal/security"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Reject cross-origin upgrades. Default gorilla CheckOrigin (and the
	// prior `return true`) made the PTY WebSocket reachable from any web
	// page the user visited — classic CSWSH into a full shell.
	CheckOrigin: func(r *http.Request) bool {
		return security.AllowedOrigin(r.Header.Get("Origin"), r.Host)
	},
}

// HandleTerminalWebSocket handles WebSocket connections for terminal I/O.
// Authentication: the URL must carry `?token=<session-bearer>`. Browsers
// don't allow setting custom headers on WebSocket connections, so a query
// param is the practical option; Origin checking (in upgrader) blocks
// cross-origin pages from grabbing it via XHR.
func HandleTerminalWebSocket(c *gin.Context) {
	sessionID := c.Param("id")
	token := c.Query("token")

	session, err := GetManager().Authenticate(sessionID, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session token"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer func() {
		conn.Close()
		session.mu.Lock()
		session.WSConn = nil
		session.mu.Unlock()
	}()

	// Store WebSocket connection in session for sending editor actions
	session.mu.Lock()
	session.WSConn = conn
	session.mu.Unlock()

	// Channel to signal shutdown (use sync.Once to prevent double close)
	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() {
		closeOnce.Do(func() {
			close(done)
		})
	}

	// Read from PTY and send to WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
				n, err := session.PTY.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("PTY read error: %v", err)
					}
					closeDone()
					return
				}
				if n > 0 {
					msg := WSMessage{
						Type: "output",
						Data: string(buf[:n]),
					}
					if err := conn.WriteJSON(msg); err != nil {
						log.Printf("WebSocket write error: %v", err)
						return
					}
				}
			}
		}
	}()

	// Read from WebSocket and handle messages
	for {
		select {
		case <-done:
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				closeDone()
				return
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				log.Printf("Failed to parse WebSocket message: %v", err)
				continue
			}

			switch wsMsg.Type {
			case "input":
				handleInput(session, wsMsg.Data)
			case "resize":
				handleResize(session, wsMsg.Data)
			case "editor_update":
				handleEditorUpdate(session, wsMsg.Data)
			}
		}
	}
}

func handleInput(session *Session, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	input, ok := dataMap["data"].(string)
	if !ok {
		return
	}

	if session.PTY != nil {
		session.PTY.Write([]byte(input))
	}
}

func handleResize(session *Session, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	cols, colsOk := dataMap["cols"].(float64)
	rows, rowsOk := dataMap["rows"].(float64)

	if colsOk && rowsOk {
		GetManager().ResizePTY(session.ID, int(cols), int(rows))
	}
}

func handleEditorUpdate(session *Session, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	content, _ := dataMap["content"].(string)

	state := &EditorState{
		Content: content,
	}

	// Parse optional selection
	if selData, ok := dataMap["selection"].(map[string]interface{}); ok {
		start, startOk := selData["start"].(float64)
		end, endOk := selData["end"].(float64)
		if startOk && endOk {
			state.Selection = &Selection{
				Start: int(start),
				End:   int(end),
			}
		}
	}

	// Parse optional cursor
	if cursorData, ok := dataMap["cursor"].(map[string]interface{}); ok {
		line, lineOk := cursorData["line"].(float64)
		column, colOk := cursorData["column"].(float64)
		if lineOk && colOk {
			state.Cursor = &Position{
				Line:   int(line),
				Column: int(column),
			}
		}
	}

	GetManager().UpdateEditorState(session.ID, state)
}
