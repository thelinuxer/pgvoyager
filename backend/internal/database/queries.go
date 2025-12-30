package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/thelinuxer/pgvoyager/internal/models"
	"github.com/google/uuid"
)

var (
	queryManager     *SavedQueryManager
	queryManagerOnce sync.Once
)

type SavedQueryManager struct {
	mu         sync.RWMutex
	queries    map[string]*models.SavedQuery
	configPath string
}

func GetQueryManager() *SavedQueryManager {
	queryManagerOnce.Do(func() {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = os.TempDir()
		}
		pgvoyagerDir := filepath.Join(configDir, "pgvoyager")
		os.MkdirAll(pgvoyagerDir, 0755)

		queryManager = &SavedQueryManager{
			queries:    make(map[string]*models.SavedQuery),
			configPath: filepath.Join(pgvoyagerDir, "queries.json"),
		}
		queryManager.loadQueries()
	})
	return queryManager
}

func (m *SavedQueryManager) loadQueries() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var queries []*models.SavedQuery
	if err := json.Unmarshal(data, &queries); err != nil {
		return err
	}

	for _, q := range queries {
		m.queries[q.ID] = q
	}
	return nil
}

func (m *SavedQueryManager) saveQueries() error {
	m.mu.RLock()
	queries := make([]*models.SavedQuery, 0, len(m.queries))
	for _, q := range m.queries {
		queries = append(queries, q)
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0600)
}

func (m *SavedQueryManager) List() []*models.SavedQuery {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.SavedQuery, 0, len(m.queries))
	for _, q := range m.queries {
		result = append(result, q)
	}
	return result
}

func (m *SavedQueryManager) Get(id string) (*models.SavedQuery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	q, ok := m.queries[id]
	if !ok {
		return nil, fmt.Errorf("saved query not found: %s", id)
	}
	return q, nil
}

func (m *SavedQueryManager) Create(req *models.SavedQueryRequest) (*models.SavedQuery, error) {
	q := &models.SavedQuery{
		ID:           uuid.New().String(),
		Name:         req.Name,
		SQL:          req.SQL,
		ConnectionID: req.ConnectionID,
		Description:  req.Description,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	m.mu.Lock()
	m.queries[q.ID] = q
	m.mu.Unlock()

	if err := m.saveQueries(); err != nil {
		return nil, err
	}

	return q, nil
}

func (m *SavedQueryManager) Update(id string, req *models.SavedQueryRequest) (*models.SavedQuery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.queries[id]
	if !ok {
		return nil, fmt.Errorf("saved query not found: %s", id)
	}

	q.Name = req.Name
	q.SQL = req.SQL
	q.ConnectionID = req.ConnectionID
	q.Description = req.Description
	q.UpdatedAt = time.Now()

	if err := m.saveQueries(); err != nil {
		return nil, err
	}

	return q, nil
}

func (m *SavedQueryManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.queries[id]; !ok {
		return fmt.Errorf("saved query not found: %s", id)
	}

	delete(m.queries, id)
	return m.saveQueries()
}
