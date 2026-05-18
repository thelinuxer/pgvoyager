// Command desktop wraps the PgVoyager HTTP server in a desktop window
// driven by an existing Chrome/Edge install via the DevTools protocol
// (lorca). Pure Go — no CGO, no platform webview SDK, no per-OS dev
// headers. Cross-compiles from a single Linux runner exactly like the
// headless `cmd/server` binary.
//
// Requires Chrome, Chromium, or Edge installed on the user's machine.
// Falls back to a clear error message if no compatible browser is found.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zserge/lorca"

	"github.com/thelinuxer/pgvoyager/internal/api"
	"github.com/thelinuxer/pgvoyager/internal/security"
	"github.com/thelinuxer/pgvoyager/internal/static"
	"github.com/thelinuxer/pgvoyager/web"
)

func main() {
	port, err := strconv.Atoi(envOr("PGVOYAGER_PORT", "0"))
	if err != nil || port < 0 || port > 65535 {
		log.Fatalf("invalid PGVOYAGER_PORT: %v", err)
	}

	host := security.ListenHost()
	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	resolved := listener.Addr().(*net.TCPAddr)
	backendURL := fmt.Sprintf("http://%s", net.JoinHostPort(host, strconv.Itoa(resolved.Port)))
	// Spawned MCP subprocesses read these to find the backend.
	_ = os.Setenv("PGVOYAGER_PORT", strconv.Itoa(resolved.Port))
	_ = os.Setenv("PGVOYAGER_BACKEND_URL", backendURL)

	gin.SetMode(gin.ReleaseMode)
	r := buildRouter()
	srv := &http.Server{
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("PgVoyager backend listening on %s", backendURL)
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Open the window. lorca launches the user's Chrome/Edge/Chromium in
	// `--app` mode against a temporary user-data dir, then drives it via
	// the DevTools protocol over a local socket.
	// Chrome 124+ tightened DevTools to require an explicit
	// `--remote-allow-origins`. lorca v0.1.10 predates that change, so
	// we pass the flag here. `*` is safe because the DevTools port
	// lorca opens is itself loopback-only.
	//
	// `--class=PgVoyager` sets WM_CLASS on Linux so the installed
	// .desktop entry (StartupWMClass=PgVoyager) matches and the dock /
	// taskbar shows the PgVoyager elephant icon instead of the generic
	// Chrome icon. The flag is harmless on macOS/Windows.
	// lorca already passes its own `--user-data-dir` into a temp dir,
	// so we don't need to manage profile isolation ourselves.
	ui, err := lorca.New(backendURL+"/", "", 1280, 800,
		"--class=PgVoyager",
		"--remote-allow-origins=*",
		"--disable-translate",
		"--disable-features=TranslateUI",
	)
	if err != nil {
		shutdown(srv)
		log.Fatalf("open window: %v (install Chrome, Chromium, or Edge — lorca drives an existing browser, it doesn't bundle one)", err)
	}
	defer ui.Close()

	// Bridge OS signals + window-close so either path triggers an
	// orderly server shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ui.Done():
	case <-sigCh:
	}
	shutdown(srv)
}

func buildRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Printf("panic recovered: %v", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}))
	r.Use(gin.Logger())
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("trusted proxies: %v", err)
	}
	r.Use(security.SecurityHeaders())
	r.Use(security.MaxBodyBytes(security.MaxRequestBodyBytes))
	r.Use(security.OriginGuard())
	// The desktop bundle is single-origin (the lorca window navigates
	// directly to the loopback server) so CORS isn't strictly needed,
	// but keeping the dev allowlist lets developers `npm run dev` the
	// frontend against a running desktop binary.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     security.DevOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Claude-Session-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(static.ServeEmbedded(web.StaticFiles, "dist"))
	api.RegisterRoutes(r)
	return r
}

func shutdown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
