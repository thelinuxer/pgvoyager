// Command desktop wraps the PgVoyager HTTP server in a native desktop
// window using Wails. The strategy is deliberately minimal: the existing
// Gin server is started in-process on the loopback interface, then the
// Wails webview navigates to it on launch. Every API, asset, and
// WebSocket route is served by the normal Gin handler — no Wails
// bindings, no duplicated code paths.
//
// Build requirements (Linux): libgtk-3-dev + libwebkit2gtk-4.0-dev (or
// 4.1-dev on newer distros). Run `wails doctor` to verify.
//
// Build: `make desktop-build` (produces ./bin/pgvoyager-desktop).
// Dev:   `make desktop-dev`   (auto-reloads the Wails wrapper).
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/thelinuxer/pgvoyager/internal/api"
	"github.com/thelinuxer/pgvoyager/internal/security"
	"github.com/thelinuxer/pgvoyager/internal/static"
	"github.com/thelinuxer/pgvoyager/web"
)

//go:embed assets/*
var assets embed.FS

func main() {
	port, err := strconv.Atoi(envOr("PGVOYAGER_PORT", "0"))
	if err != nil || port < 0 || port > 65535 {
		log.Fatalf("invalid PGVOYAGER_PORT: %v", err)
	}

	// Bind loopback. Port 0 asks the OS for a free port — appropriate for
	// the desktop bundle since users don't need to know the port.
	host := security.ListenHost()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	resolved := listener.Addr().(*net.TCPAddr)
	backendURL := fmt.Sprintf("http://%s", net.JoinHostPort(host, strconv.Itoa(resolved.Port)))
	// Surface the port for spawned MCP subprocesses, which read
	// PGVOYAGER_BACKEND_URL / PGVOYAGER_PORT from env.
	_ = os.Setenv("PGVOYAGER_PORT", strconv.Itoa(resolved.Port))
	_ = os.Setenv("PGVOYAGER_BACKEND_URL", backendURL)

	gin.SetMode(gin.ReleaseMode)
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
	// The desktop bundle is single-origin (Wails webview navigates to
	// the loopback HTTP server). CORS isn't needed; keep a permissive
	// OPTIONS responder only so any future browser dev tooling works.
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

	// Wails AssetServer answers the initial wails://wails/index.html load
	// with a tiny shim that navigates the webview to the local backend.
	// From that point on every request originates from
	// http://127.0.0.1:<port> and is served by Gin — Wails is just the
	// window chrome.
	bootstrapHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>PgVoyager</title>
<style>html,body{background:#1e1e2e;color:#cdd6f4;font-family:system-ui;margin:0;display:flex;align-items:center;justify-content:center;height:100vh}</style>
<script>window.location.replace(%q)</script>
<p>Loading PgVoyager…</p>`, backendURL+"/")
	})

	err = wails.Run(&options.App{
		Title:            "PgVoyager",
		Width:            1280,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 46, A: 1},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: bootstrapHandler,
		},
		OnShutdown: func(_ context.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		},
	})
	if err != nil {
		log.Fatalf("wails: %v", err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
