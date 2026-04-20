package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thelinuxer/pgvoyager/internal/models"
	"github.com/thelinuxer/pgvoyager/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	manager     *ConnectionManager
	managerOnce sync.Once
)

type ConnectionManager struct {
	mu          sync.RWMutex
	connections map[string]*models.Connection
	pools       map[string]*pgxpool.Pool
}

func GetManager() *ConnectionManager {
	managerOnce.Do(func() {
		manager = &ConnectionManager{
			connections: make(map[string]*models.Connection),
			pools:       make(map[string]*pgxpool.Pool),
		}
		manager.loadConnections()
	})
	return manager
}

func (m *ConnectionManager) loadConnections() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, err := storage.GetDB()
	if err != nil {
		return err
	}

	rows, err := db.Query(`
		SELECT id, name, host, port, database, username, password, ssl_mode, created_at
		FROM connections
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		conn := &models.Connection{}
		err := rows.Scan(
			&conn.ID,
			&conn.Name,
			&conn.Host,
			&conn.Port,
			&conn.Database,
			&conn.Username,
			&conn.Password,
			&conn.SSLMode,
			&conn.CreatedAt,
		)
		if err != nil {
			return err
		}
		conn.IsConnected = false
		conn.UpdatedAt = conn.CreatedAt
		m.connections[conn.ID] = conn
	}
	return rows.Err()
}

func (m *ConnectionManager) List() []*models.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		connCopy := *conn
		connCopy.Password = "" // Don't expose password
		result = append(result, &connCopy)
	}
	return result
}

func (m *ConnectionManager) Get(id string) (*models.Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[id]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	connCopy := *conn
	connCopy.Password = ""
	return &connCopy, nil
}

// GetWithPassword retrieves a connection including the password (for internal use only)
func (m *ConnectionManager) GetWithPassword(id string) (*models.Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[id]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	connCopy := *conn
	return &connCopy, nil
}

func (m *ConnectionManager) Create(req *models.ConnectionRequest) (*models.Connection, error) {
	database := req.Database
	if database == "" {
		database = models.DefaultDatabase
	}

	conn := &models.Connection{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Host:      req.Host,
		Port:      req.Port,
		Database:  database,
		Username:  req.Username,
		Password:  req.Password,
		SSLMode:   req.SSLMode,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if conn.SSLMode == "" {
		conn.SSLMode = "prefer"
	}

	db, err := storage.GetDB()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		INSERT INTO connections (id, name, host, port, database, username, password, ssl_mode, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, conn.ID, conn.Name, conn.Host, conn.Port, conn.Database, conn.Username, conn.Password, conn.SSLMode, conn.CreatedAt)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.connections[conn.ID] = conn
	m.mu.Unlock()

	connCopy := *conn
	connCopy.Password = ""
	return &connCopy, nil
}

func (m *ConnectionManager) Update(id string, req *models.ConnectionRequest) (*models.Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[id]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	conn.Name = req.Name
	conn.Host = req.Host
	conn.Port = req.Port
	if req.Database != "" {
		conn.Database = req.Database
	} else if conn.Database == "" {
		conn.Database = models.DefaultDatabase
	}
	conn.Username = req.Username
	if req.Password != "" {
		conn.Password = req.Password
	}
	conn.SSLMode = req.SSLMode
	conn.UpdatedAt = time.Now()

	db, err := storage.GetDB()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		UPDATE connections
		SET name = ?, host = ?, port = ?, database = ?, username = ?, password = ?, ssl_mode = ?
		WHERE id = ?
	`, conn.Name, conn.Host, conn.Port, conn.Database, conn.Username, conn.Password, conn.SSLMode, id)
	if err != nil {
		return nil, err
	}

	connCopy := *conn
	connCopy.Password = ""
	return &connCopy, nil
}

func (m *ConnectionManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.connections[id]; !ok {
		return fmt.Errorf("connection not found: %s", id)
	}

	// Disconnect if connected
	if pool, ok := m.pools[id]; ok {
		pool.Close()
		delete(m.pools, id)
	}

	db, err := storage.GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM connections WHERE id = ?", id)
	if err != nil {
		return err
	}

	delete(m.connections, id)
	return nil
}

func (m *ConnectionManager) buildConnString(conn *models.Connection) string {
	database := conn.Database
	if database == "" {
		database = models.DefaultDatabase
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		conn.Username,
		conn.Password,
		conn.Host,
		conn.Port,
		database,
		conn.SSLMode,
	)
}

func (m *ConnectionManager) TestConnection(req *models.TestConnectionRequest) error {
	database := req.Database
	if database == "" {
		database = models.DefaultDatabase
	}
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		req.Username,
		req.Password,
		req.Host,
		req.Port,
		database,
		req.SSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a minimal pool configuration for testing
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return err
	}
	config.MaxConns = 1 // Only need one connection for testing
	config.MinConns = 0

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	defer pool.Close()

	return pool.Ping(ctx)
}

func (m *ConnectionManager) Connect(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[id]
	if !ok {
		return fmt.Errorf("connection not found: %s", id)
	}

	if _, ok := m.pools[id]; ok {
		return nil // Already connected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure pool with limited connections to avoid exhausting PostgreSQL
	config, err := pgxpool.ParseConfig(m.buildConnString(conn))
	if err != nil {
		return err
	}
	config.MaxConns = 5                          // Limit max connections per pool
	config.MinConns = 0                          // Don't keep idle connections
	config.MaxConnIdleTime = 5 * time.Minute     // Close idle connections faster
	config.MaxConnLifetime = 30 * time.Minute    // Recycle connections

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return err
	}

	m.pools[id] = pool
	conn.IsConnected = true
	return nil
}

func (m *ConnectionManager) Disconnect(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[id]
	if !ok {
		return fmt.Errorf("connection not found: %s", id)
	}

	if pool, ok := m.pools[id]; ok {
		pool.Close()
		delete(m.pools, id)
	}

	conn.IsConnected = false
	return nil
}

func (m *ConnectionManager) GetPool(id string) (*pgxpool.Pool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, ok := m.pools[id]
	if !ok {
		return nil, fmt.Errorf("not connected: %s", id)
	}
	return pool, nil
}

func (m *ConnectionManager) IsConnected(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.pools[id]
	return ok
}

// SwitchDatabase reopens the connection's pool against a different database on the same server.
// The new database name is persisted so reconnects target the last-selected database.
func (m *ConnectionManager) SwitchDatabase(id, dbName string) (*models.Connection, error) {
	if dbName == "" {
		return nil, fmt.Errorf("database name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[id]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	if conn.Database == dbName {
		if pool, ok := m.pools[id]; ok && pool != nil {
			connCopy := *conn
			connCopy.Password = ""
			return &connCopy, nil
		}
	}

	previousDB := conn.Database
	conn.Database = dbName
	conn.UpdatedAt = time.Now()

	if oldPool, ok := m.pools[id]; ok {
		oldPool.Close()
		delete(m.pools, id)
		conn.IsConnected = false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(m.buildConnString(conn))
	if err != nil {
		conn.Database = previousDB
		return nil, err
	}
	config.MaxConns = 5
	config.MinConns = 0
	config.MaxConnIdleTime = 5 * time.Minute
	config.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		conn.Database = previousDB
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		conn.Database = previousDB
		return nil, err
	}

	db, err := storage.GetDB()
	if err != nil {
		pool.Close()
		conn.Database = previousDB
		return nil, err
	}

	if _, err := db.Exec(`UPDATE connections SET database = ? WHERE id = ?`, dbName, id); err != nil {
		pool.Close()
		conn.Database = previousDB
		return nil, err
	}

	m.pools[id] = pool
	conn.IsConnected = true

	connCopy := *conn
	connCopy.Password = ""
	return &connCopy, nil
}

// pgQuoteIdent quotes a Postgres identifier for safe inclusion in DDL.
// Postgres doesn't support parameter binding for database/role names, so we
// must hand-quote them. Double any embedded `"` and wrap in `"`.
func pgQuoteIdent(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			out = append(out, '"', '"')
		} else {
			out = append(out, s[i])
		}
	}
	out = append(out, '"')
	return string(out)
}

// CreateDatabase issues CREATE DATABASE on the server via the connection's
// current pool. Requires CREATEDB privilege.
func (m *ConnectionManager) CreateDatabase(id string, req *models.CreateDatabaseRequest) error {
	pool, err := m.GetPool(id)
	if err != nil {
		return err
	}

	sql := "CREATE DATABASE " + pgQuoteIdent(req.Name)
	if req.Owner != "" {
		sql += " OWNER " + pgQuoteIdent(req.Owner)
	}
	if req.Template != "" {
		sql += " TEMPLATE " + pgQuoteIdent(req.Template)
	}
	if req.Encoding != "" {
		sql += " ENCODING " + pgQuoteIdent(req.Encoding)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = pool.Exec(ctx, sql)
	return err
}

// DropDatabase issues DROP DATABASE on the server. If the target is the
// currently-selected database for the connection, it auto-switches to
// `postgres` first. If force is true, active sessions on the target are
// terminated before the drop.
func (m *ConnectionManager) DropDatabase(id, dbName string, force bool) error {
	if dbName == "" {
		return fmt.Errorf("database name is required")
	}

	m.mu.RLock()
	conn, ok := m.connections[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("connection not found: %s", id)
	}

	// Can't drop the DB we're connected to — switch away first.
	if conn.Database == dbName {
		fallback := models.DefaultDatabase
		if fallback == dbName {
			return fmt.Errorf("cannot drop the default maintenance database `%s`", dbName)
		}
		if _, err := m.SwitchDatabase(id, fallback); err != nil {
			return fmt.Errorf("switch to %s before drop failed: %w", fallback, err)
		}
	}

	pool, err := m.GetPool(id)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if force {
		terminateSQL := `
			SELECT pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = $1 AND pid <> pg_backend_pid()
		`
		if _, err := pool.Exec(ctx, terminateSQL, dbName); err != nil {
			return fmt.Errorf("terminate active sessions: %w", err)
		}
	}

	if _, err := pool.Exec(ctx, "DROP DATABASE "+pgQuoteIdent(dbName)); err != nil {
		return err
	}
	return nil
}
