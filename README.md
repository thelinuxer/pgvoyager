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

- **Go 1.24+** - Backend runtime
- **Node.js 20+** - Frontend toolchain
- **Claude Code CLI** - For the embedded AI assistant (`npm install -g @anthropic-ai/claude-code` or see [Claude Code docs](https://docs.anthropic.com/en/docs/claude-code))
- **PostgreSQL** - A database to connect to

### Installation

```bash
# Clone the repo
git clone https://github.com/thelinuxer/pgvoyager.git
cd pgvoyager

# Install dependencies
make install

# Build everything (backend + MCP server + frontend)
make build
```

### Running

**Development mode** (hot reload):
```bash
make dev
```
This starts:
- Backend API on `http://localhost:8081`
- Frontend on `http://localhost:5173`

**Production build**:
```bash
# After running make build
./bin/pgvoyager  # Start the backend
cd frontend && npm run preview  # Serve the built frontend
```

### First Steps

1. Open `http://localhost:5173` in your browser
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
