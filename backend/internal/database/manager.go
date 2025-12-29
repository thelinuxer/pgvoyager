package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/atoulan/pgvoyager/internal/models"
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
	configPath  string
}

func GetManager() *ConnectionManager {
	managerOnce.Do(func() {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = os.TempDir()
		}
		pgvoyagerDir := filepath.Join(configDir, "pgvoyager")
		os.MkdirAll(pgvoyagerDir, 0755)

		manager = &ConnectionManager{
			connections: make(map[string]*models.Connection),
			pools:       make(map[string]*pgxpool.Pool),
			configPath:  filepath.Join(pgvoyagerDir, "connections.json"),
		}
		manager.loadConnections()
	})
	return manager
}

func (m *ConnectionManager) loadConnections() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var connections []*models.Connection
	if err := json.Unmarshal(data, &connections); err != nil {
		return err
	}

	for _, conn := range connections {
		conn.IsConnected = false
		m.connections[conn.ID] = conn
	}
	return nil
}

func (m *ConnectionManager) saveConnections() error {
	m.mu.RLock()
	connections := make([]*models.Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		// Don't save password in plain text - this should use secure storage
		connCopy := *conn
		connections = append(connections, &connCopy)
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(connections, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0600)
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

	m.mu.Lock()
	m.connections[conn.ID] = conn
	m.mu.Unlock()

	if err := m.saveConnections(); err != nil {
		return nil, err
	}

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

	if err := m.saveConnections(); err != nil {
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

	delete(m.connections, id)
	return m.saveConnections()
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
