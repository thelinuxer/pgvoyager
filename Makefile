.PHONY: all backend frontend dev dev-e2e backend-e2e install clean build build-prod release \
	e2e-install e2e-test e2e-test-headed e2e-test-ui e2e-smoke e2e-tier1 e2e-tier2 e2e-tier3 e2e-report

# Version from git tag or default
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/thelinuxer/pgvoyager/internal/version.Version=$(VERSION)

# Default target: run both backend and frontend
dev:
	@echo "Starting PgVoyager..."
	@echo "App available at http://localhost:5137"
	@make -j2 backend frontend

# Run for e2e tests with isolated config directory (doesn't affect your real config)
dev-e2e:
	@echo "Starting PgVoyager for e2e tests (isolated config)..."
	@echo "App available at http://localhost:5137"
	@echo "Config directory: /tmp/pgvoyager-e2e-config"
	@make -j2 backend-e2e frontend

# Run backend for e2e tests with isolated config
backend-e2e:
	@echo "Starting Go backend on port 5138 (e2e mode)..."
	cd backend && PGVOYAGER_PORT=5138 PGVOYAGER_CONFIG_DIR=/tmp/pgvoyager-e2e-config go run ./cmd/server

# Run backend in development mode (on port 5138, proxied by frontend)
backend:
	@echo "Starting Go backend on port 5138..."
	cd backend && PGVOYAGER_PORT=5138 go run ./cmd/server

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
	cd backend && go build -ldflags="$(LDFLAGS)" -o ../bin/pgvoyager ./cmd/server
	@echo "Building MCP server..."
	cd backend && go build -ldflags="$(LDFLAGS)" -o ../bin/pgvoyager-mcp ./cmd/mcp
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

# ====================
# E2E Testing Commands
# ====================

# Install E2E test dependencies
e2e-install:
	@echo "Installing E2E test dependencies..."
	cd e2e && npm install
	cd e2e && npx playwright install chromium --with-deps

# Run all E2E tests (headless)
e2e-test:
	@echo "Running all E2E tests..."
	cd e2e && npm test

# Run E2E tests in headed mode (visible browser)
e2e-test-headed:
	@echo "Running E2E tests in headed mode..."
	cd e2e && npm run test:headed

# Run E2E tests with Playwright UI
e2e-test-ui:
	@echo "Opening Playwright UI..."
	cd e2e && npm run test:ui

# Run smoke tests only
e2e-smoke:
	@echo "Running smoke tests..."
	cd e2e && npm run test:smoke

# Run Tier 1 (critical) tests
e2e-tier1:
	@echo "Running Tier 1 (critical) tests..."
	cd e2e && npm run test:tier1

# Run Tier 2 (important) tests
e2e-tier2:
	@echo "Running Tier 2 (important) tests..."
	cd e2e && npm run test:tier2

# Run Tier 3 (extended) tests
e2e-tier3:
	@echo "Running Tier 3 (extended) tests..."
	cd e2e && npm run test:tier3

# Open test report
e2e-report:
	@echo "Opening test report..."
	cd e2e && npm run report
