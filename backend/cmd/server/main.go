package main

import (
	"context"
	"errors"
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
	"github.com/thelinuxer/pgvoyager/internal/api"
	"github.com/thelinuxer/pgvoyager/internal/security"
	"github.com/thelinuxer/pgvoyager/internal/static"
	"github.com/thelinuxer/pgvoyager/web"
)

func main() {
	port := os.Getenv("PGVOYAGER_PORT")
	if port == "" {
		port = "5137"
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		log.Fatalf("invalid PGVOYAGER_PORT %q: must be an integer in [1, 65535]", port)
	}

	host := security.ListenHost()
	isProd := os.Getenv("PGVOYAGER_MODE") == "production"

	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// Custom recovery: log the panic but return a generic 500 (no stack trace
	// in the response body, ever — gin.Default's recovery leaks the stack in
	// non-release mode).
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Printf("panic recovered: %v", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}))
	r.Use(gin.Logger())
	// Trust no upstream proxies — PgVoyager binds loopback by default; any
	// X-Forwarded-* header from a client is forged.
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("failed to clear trusted proxies: %v", err)
	}
	r.Use(security.SecurityHeaders())
	r.Use(security.MaxBodyBytes(security.MaxRequestBodyBytes))

	if isProd {
		r.Use(static.ServeEmbedded(web.StaticFiles, "dist"))
		log.Printf("PgVoyager running in production mode")
	} else {
		// Dev: SvelteKit runs on a different port. AllowCredentials is
		// false — the API uses no cookies / Authorization headers, so
		// credentialed CORS would just widen attack surface.
		r.Use(cors.New(cors.Config{
			AllowOrigins:     security.DevOrigins(),
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "X-Claude-Session-ID"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: false,
			MaxAge:           12 * time.Hour,
		}))
		log.Printf("PgVoyager running in development mode (CORS enabled)")
	}

	api.RegisterRoutes(r)

	addr := net.JoinHostPort(host, port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
		// Timeouts mitigate Slowloris and slow-body DoS. WriteTimeout
		// is generous (60s) because some schema queries on large DBs
		// can be slow; the per-handler context timeouts cap actual SQL.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("PgVoyager server starting on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Printf("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
