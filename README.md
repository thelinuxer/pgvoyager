<p align="center">
  <img src="frontend/static/logo.svg" alt="PgVoyager" width="120">
</p>

# PgVoyager

A PostgreSQL database explorer with an embedded Claude Code terminal. Built entirely through vibe coding with Claude.

## What is Vibe Coding?

This project was created through conversational development with Claude - describing what I wanted, iterating on ideas, and letting AI handle the implementation details. No traditional IDE-driven development, just vibes and good prompts.

## What It Does

PgVoyager is a modern PostgreSQL client that lets you:

- **Explore your database** - Browse schemas, tables, views, functions, indexes, and foreign keys
- **Write and run SQL** - Full-featured query editor with syntax highlighting, autocomplete, and error highlighting
- **Track query history** - See past queries with execution times and success/failure status
- **Ask Claude for help** - Embedded Claude Code terminal with MCP integration that can:
  - Explore your database schema
  - Write SQL queries directly into the editor
  - Execute queries and analyze results
  - Help you understand your data

```
┌─────────────────────────────────────────────────────────────────────┐
│  PgVoyager                                                          │
├──────────────────────────────────┬──────────────────────────────────┤
│  Schema Browser                  │  Query Editor                    │
│  ├── public                      │  ┌────────────────────────────┐  │
│  │   ├── users                   │  │ SELECT * FROM users        │  │
│  │   ├── orders                  │  │ WHERE created_at > now()   │  │
│  │   └── products                │  │   - interval '7 days';     │  │
│  └── analytics                   │  └────────────────────────────┘  │
│      └── events                  │  ┌────────────────────────────┐  │
│                                  │  │ Results: 42 rows           │  │
│                                  │  └────────────────────────────┘  │
├──────────────────────────────────┴──────────────────────────────────┤
│  Claude Terminal                                                     │
│  > What tables have the most rows?                                   │
│  I'll check the row counts for you...                                │
└─────────────────────────────────────────────────────────────────────┘
```

## Getting Started

### Prerequisites

- **Claude Code CLI** - For the embedded AI assistant (`npm install -g @anthropic-ai/claude-code` or see [Claude Code docs](https://docs.anthropic.com/en/docs/claude-code))
- **PostgreSQL** - A database to connect to

### Installation

Download the latest release for your platform from [GitHub Releases](https://github.com/thelinuxer/pgvoyager/releases):

#### Linux (Recommended)

Download the packaged release with desktop integration:

```bash
# Download and extract
curl -L https://github.com/thelinuxer/pgvoyager/releases/latest/download/pgvoyager-linux-amd64.tar.gz | tar xz
cd pgvoyager-linux-amd64

# Run the installer (adds app icon and desktop entry)
./install.sh

# Launch from your application menu, or run:
pgvoyager-launcher
```

#### Windows

Download `pgvoyager-windows-amd64.zip` from the releases page, extract it, and run:

```powershell
# Run the installer (adds Start Menu shortcut)
.\install.ps1

# Or just double-click pgvoyager-launcher.bat
```

#### macOS / Raw Binaries

For macOS or if you prefer raw binaries without installers:

| Platform | Binary |
|----------|--------|
| Linux (x64) | `pgvoyager-linux-amd64` |
| Linux (ARM64) | `pgvoyager-linux-arm64` |
| macOS (Intel) | `pgvoyager-darwin-amd64` |
| macOS (Apple Silicon) | `pgvoyager-darwin-arm64` |
| Windows | `pgvoyager-windows-amd64.exe` |

```bash
# Download and run directly
curl -L https://github.com/thelinuxer/pgvoyager/releases/latest/download/pgvoyager-darwin-arm64 -o pgvoyager
chmod +x pgvoyager
PGVOYAGER_MODE=production ./pgvoyager
```

Then open `http://localhost:8081` in your browser.

### Building from Source

**Prerequisites for building:**
- Go 1.24+
- Node.js 20+

```bash
# Clone the repo
git clone https://github.com/thelinuxer/pgvoyager.git
cd pgvoyager

# Install dependencies
make install

# Build production binary with embedded frontend
make build-prod

# Run it
PGVOYAGER_MODE=production ./bin/pgvoyager
```

### Development Mode

For development with hot reload:

```bash
make dev
```

This starts:
- Backend API on `http://localhost:8081`
- Frontend on `http://localhost:5173`

### First Steps

1. Open your browser:
   - Production: `http://localhost:8081`
   - Development: `http://localhost:5173`
2. Click "New Connection" and enter your PostgreSQL credentials
3. Browse your schemas in the left sidebar
4. Open a query tab and write some SQL
5. Click the Claude icon to open the AI assistant

## Architecture

```
┌─────────────────────┐     WebSocket      ┌─────────────────────┐
│  Frontend (Svelte)  │ <───────────────── │  Backend (Go)       │
│  - Schema browser   │                    │  - REST API         │
│  - Query editor     │                    │  - PTY Manager      │
│  - xterm.js         │                    │  - WebSocket        │
└─────────────────────┘                    └──────────┬──────────┘
                                                      │ PTY
                                                      ▼
                                           ┌─────────────────────┐
                                           │  Claude Code CLI    │
                                           │  --mcp-config       │
                                           └──────────┬──────────┘
                                                      │ MCP (stdio)
                                                      ▼
                                           ┌─────────────────────┐
                                           │  PgVoyager MCP      │
                                           │  Server             │
                                           └─────────────────────┘
```

**Key components:**

- **Frontend**: SvelteKit 2 + Svelte 5 with CodeMirror for SQL editing and xterm.js for the terminal
- **Backend**: Go with Gin framework, manages database connections and Claude sessions
- **MCP Server**: Separate Go binary that gives Claude access to database tools

## Hacking

### Project Structure

```
pgvoyager/
├── backend/
│   ├── cmd/
│   │   ├── server/          # Main API server
│   │   └── mcp/             # MCP server binary
│   └── internal/
│       ├── api/             # Route definitions
│       ├── handlers/        # HTTP handlers
│       ├── database/        # Connection pool management
│       └── claude/          # PTY session & MCP integration
├── frontend/
│   └── src/
│       ├── routes/          # SvelteKit pages
│       └── lib/
│           ├── components/  # UI components
│           ├── stores/      # Svelte stores (state)
│           └── api/         # API client
├── bin/                     # Built binaries
└── Makefile
```

### Development Workflow

```bash
# Run in dev mode with hot reload
make dev

# Run just the backend
make run-backend

# Run just the frontend
make run-frontend

# Build specific components
make build-backend
make build-mcp
make build-frontend

# Clean build artifacts
make clean
```

### Key Files

| File | Purpose |
|------|---------|
| `backend/internal/claude/manager.go` | PTY session lifecycle, spawns Claude Code |
| `backend/internal/claude/websocket.go` | Terminal I/O over WebSocket |
| `backend/cmd/mcp/main.go` | MCP server with database tools |
| `frontend/src/lib/components/ClaudeTerminalPanel.svelte` | xterm.js terminal UI |
| `frontend/src/lib/stores/claudeTerminal.ts` | Terminal WebSocket state |
| `frontend/src/lib/stores/editor.ts` | Shared editor state for Claude integration |

### MCP Tools

The MCP server exposes these tools to Claude:

| Tool | Description |
|------|-------------|
| `get_connection_info` | Current database connection details |
| `list_schemas` | All schemas in the database |
| `list_tables` | Tables (optionally filtered by schema) |
| `get_columns` | Column details for a table |
| `get_table_info` | Table size, row count, etc. |
| `execute_query` | Run arbitrary SQL |
| `list_views` | Database views |
| `list_functions` | Database functions |
| `get_foreign_keys` | Foreign key relationships |
| `get_indexes` | Index information |
| `get_editor_content` | Read SQL from the query editor |
| `insert_to_editor` | Insert text into the editor |
| `replace_editor_content` | Replace editor content |

### Adding New MCP Tools

1. Add the handler in `backend/internal/handlers/mcp.go`
2. Add the route in `backend/internal/api/routes.go`
3. Add the tool definition in `backend/cmd/mcp/main.go`
4. Add to the allowed tools list in `backend/internal/claude/manager.go`

### Tech Stack

**Backend:**
- Go 1.24
- Gin (HTTP framework)
- pgx (PostgreSQL driver)
- creack/pty (pseudo-terminal)
- gorilla/websocket
- mcp-go (MCP protocol)

**Frontend:**
- Svelte 5 + SvelteKit 2
- TypeScript
- CodeMirror 6 (SQL editor)
- xterm.js (terminal emulator)
- Vite 7

## License

MIT
