package main

import (
	"log"
	"os"
	"time"

	"github.com/thelinuxer/pgvoyager/internal/api"
	"github.com/thelinuxer/pgvoyager/internal/static"
	"github.com/thelinuxer/pgvoyager/web"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PGVOYAGER_PORT")
	if port == "" {
		port = "8081"
	}

	// Check if we're in production mode (serving embedded frontend)
	isProd := os.Getenv("PGVOYAGER_MODE") == "production"

	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	if isProd {
		// Production: serve embedded static files
		r.Use(static.ServeEmbedded(web.StaticFiles, "dist"))
		log.Printf("PgVoyager running in production mode")
	} else {
		// Development: CORS for separate frontend dev server
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
		log.Printf("PgVoyager running in development mode (CORS enabled)")
	}

	// Register API routes
	api.RegisterRoutes(r)

	log.Printf("PgVoyager server starting on http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
