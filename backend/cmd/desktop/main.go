// Command desktop wraps the PgVoyager HTTP server in a desktop window
// by launching an installed Chromium-family browser in `--app` mode
// pointing at a loopback URL. Pure Go — no CGO, no platform webview SDK,
// no per-OS dev headers. Cross-compiles from any host.
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
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/thelinuxer/pgvoyager/internal/api"
	"github.com/thelinuxer/pgvoyager/internal/chromelaunch"
	"github.com/thelinuxer/pgvoyager/internal/handlers"
	"github.com/thelinuxer/pgvoyager/internal/security"
	"github.com/thelinuxer/pgvoyager/internal/selfupdate"
	"github.com/thelinuxer/pgvoyager/internal/static"
	"github.com/thelinuxer/pgvoyager/internal/version"
	"github.com/thelinuxer/pgvoyager/web"
)

func main() {
	port, err := strconv.Atoi(envOr("PGVOYAGER_PORT", "0"))
	if err != nil || port < 0 || port > 65535 {
		log.Fatalf("invalid PGVOYAGER_PORT: %v", err)
	}

	chromePath, err := chromelaunch.Find()
	if err != nil {
		log.Fatalf("%v", err)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updater := selfupdate.NewManager(version.Version)
	handlers.SetUpdateManager(updater)
	updater.Start(ctx, 6*time.Hour)

	// Bridge OS signals into ctx-cancel so either the user closing the
	// browser window or SIGINT/SIGTERM tears down the server cleanly.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	runErr := chromelaunch.Run(ctx, chromePath, chromelaunch.Options{
		URL:         backendURL + "/",
		Width:       1280,
		Height:      800,
		AppClass:    "PgVoyager",
		DesktopFile: findInstalledDesktopFile(),
	})

	shutdown(srv)
	if runErr != nil {
		log.Fatalf("browser: %v", runErr)
	}
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
	// Dev allowlist kept so a developer can `npm run dev` the frontend
	// against a running desktop binary; the desktop bundle itself is
	// single-origin so CORS isn't required in normal use.
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
	// Desktop-only: the mutating restart route lives here so the headless
	// server binary never exposes "replace my binary" over HTTP.
	r.POST("/api/update/restart", handlers.UpdateRestart)
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

// findInstalledDesktopFile returns an absolute path to pgvoyager.desktop
// if the system installer placed one in the conventional XDG locations.
// Used by chromelaunch to set _NET_WM_DESKTOP_FILE on the spawned
// browser window so GNOME Shell attaches the correct dock icon.
// Returns "" if no .desktop entry was found — the launcher then skips
// the property-set step.
func findInstalledDesktopFile() string {
	candidates := []string{
		filepath.Join(envOr("XDG_DATA_HOME", filepath.Join(envOr("HOME", ""), ".local/share")), "applications/pgvoyager.desktop"),
		"/usr/local/share/applications/pgvoyager.desktop",
		"/usr/share/applications/pgvoyager.desktop",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
