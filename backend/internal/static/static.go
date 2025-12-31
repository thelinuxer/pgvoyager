package static

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServeEmbedded serves embedded static files with SPA fallback
func ServeEmbedded(staticFS fs.FS, subDir string) gin.HandlerFunc {
	// Get the subdirectory filesystem
	subFS, err := fs.Sub(staticFS, subDir)
	if err != nil {
		panic("failed to get sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(subFS))

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes
		if strings.HasPrefix(path, "/api") {
			c.Next()
			return
		}

		// Skip WebSocket routes
		if strings.HasPrefix(path, "/ws") {
			c.Next()
			return
		}

		// Try to serve the file
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		file, err := subFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			file.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		// SPA fallback: serve index.html for any non-file route
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
