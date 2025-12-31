.PHONY: all backend frontend dev install clean build build-prod release

# Version from git tag or default
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/thelinuxer/pgvoyager/internal/version.Version=$(VERSION)

# Default target: run both backend and frontend
dev:
	@echo "Starting PgVoyager..."
	@echo "Backend will run on http://localhost:5137"
	@echo "Frontend will run on http://localhost:5173"
	@make -j2 backend frontend

# Run backend in development mode
backend:
	@echo "Starting Go backend..."
	cd backend && go run ./cmd/server

# Run frontend in development mode
frontend:
	@echo "Starting SvelteKit frontend..."
	cd frontend && npm run dev

# Install all dependencies
install:
	@echo "Installing backend dependencies..."
	cd backend && go mod download
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

# Build for development (separate binaries)
build:
	@echo "Building backend..."
	cd backend && go build -o ../bin/pgvoyager ./cmd/server
	@echo "Building MCP server..."
	cd backend && go build -o ../bin/pgvoyager-mcp ./cmd/mcp
	@echo "Building frontend..."
	cd frontend && npm run build

# Build production single binary with embedded frontend
build-prod: build-frontend-prod
	@echo "Building production binary..."
	cd backend && CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/pgvoyager ./cmd/server
	@echo "Building MCP server..."
	cd backend && CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../bin/pgvoyager-mcp ./cmd/mcp
	@echo "Production build complete: bin/pgvoyager"

# Build frontend and copy to backend/web/dist for embedding
build-frontend-prod:
	@echo "Building frontend for production..."
	cd frontend && npm run build
	@echo "Copying frontend build to backend/web/dist..."
	rm -rf backend/web/dist/*
	cp -r frontend/build/* backend/web/dist/

# Cross-compile for all platforms
release: build-frontend-prod
	@echo "Building releases for all platforms..."
	@mkdir -p releases

	# Linux AMD64
	@echo "Building linux-amd64..."
	cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-linux-amd64 ./cmd/server
	cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-mcp-linux-amd64 ./cmd/mcp

	# Linux ARM64
	@echo "Building linux-arm64..."
	cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-linux-arm64 ./cmd/server
	cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-mcp-linux-arm64 ./cmd/mcp

	# macOS AMD64
	@echo "Building darwin-amd64..."
	cd backend && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-darwin-amd64 ./cmd/server
	cd backend && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-mcp-darwin-amd64 ./cmd/mcp

	# macOS ARM64 (Apple Silicon)
	@echo "Building darwin-arm64..."
	cd backend && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-darwin-arm64 ./cmd/server
	cd backend && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-mcp-darwin-arm64 ./cmd/mcp

	# Windows AMD64
	@echo "Building windows-amd64..."
	cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-windows-amd64.exe ./cmd/server
	cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../releases/pgvoyager-mcp-windows-amd64.exe ./cmd/mcp

	@echo "Release builds complete in releases/"
	@ls -lh releases/

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf releases/
	rm -rf frontend/.svelte-kit
	rm -rf frontend/build

# Run backend only
run-backend:
	cd backend && go run ./cmd/server

# Run frontend only
run-frontend:
	cd frontend && npm run dev

# Run production binary
run-prod:
	PGVOYAGER_MODE=production ./bin/pgvoyager

# Build backend binary
build-backend:
	cd backend && go build -o ../bin/pgvoyager ./cmd/server

# Build MCP server binary
build-mcp:
	cd backend && go build -o ../bin/pgvoyager-mcp ./cmd/mcp

# Build frontend for production
build-frontend:
	cd frontend && npm run build
