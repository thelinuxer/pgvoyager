.PHONY: all backend frontend dev install clean build

# Default target: run both backend and frontend
dev:
	@echo "Starting PgVoyager..."
	@echo "Backend will run on http://localhost:8080"
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

# Build for production
build:
	@echo "Building backend..."
	cd backend && go build -o ../bin/pgvoyager ./cmd/server
	@echo "Building MCP server..."
	cd backend && go build -o ../bin/pgvoyager-mcp ./cmd/mcp
	@echo "Building frontend..."
	cd frontend && npm run build

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf frontend/.svelte-kit
	rm -rf frontend/build

# Run backend only
run-backend:
	cd backend && go run ./cmd/server

# Run frontend only
run-frontend:
	cd frontend && npm run dev

# Build backend binary
build-backend:
	cd backend && go build -o ../bin/pgvoyager ./cmd/server

# Build MCP server binary
build-mcp:
	cd backend && go build -o ../bin/pgvoyager-mcp ./cmd/mcp

# Build frontend for production
build-frontend:
	cd frontend && npm run build
