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
	conn := &models.Connection{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Host:      req.Host,
		Port:      req.Port,
		Database:  req.Database,
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
	conn.Database = req.Database
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
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		conn.Username,
		conn.Password,
		conn.Host,
		conn.Port,
		conn.Database,
		conn.SSLMode,
	)
}

func (m *ConnectionManager) TestConnection(req *models.TestConnectionRequest) error {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		req.Username,
		req.Password,
		req.Host,
		req.Port,
		req.Database,
		req.SSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
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

	pool, err := pgxpool.New(ctx, m.buildConnString(conn))
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
