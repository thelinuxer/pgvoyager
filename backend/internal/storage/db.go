package storage

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
	"github.com/thelinuxer/pgvoyager/internal/models"
)

var (
	db     *sql.DB
	dbOnce sync.Once
)

// GetDB returns the singleton database instance
func GetDB() (*sql.DB, error) {
	var err error
	dbOnce.Do(func() {
		configDir, e := os.UserConfigDir()
		if e != nil {
			configDir = os.TempDir()
		}
		pgvoyagerDir := filepath.Join(configDir, "pgvoyager")
		os.MkdirAll(pgvoyagerDir, 0755)

		dbPath := filepath.Join(pgvoyagerDir, "pgvoyager.db")
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return
		}

		// Initialize schema
		if _, err = db.Exec(schema); err != nil {
			return
		}

		// Migrate from old connections.json if exists
		err = migrateFromJSON(pgvoyagerDir)
	})
	return db, err
}

// migrateFromJSON migrates data from old connections.json file
func migrateFromJSON(configDir string) error {
	jsonPath := filepath.Join(configDir, "connections.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No old file to migrate
		}
		return err
	}

	var connections []*models.Connection
	if err := json.Unmarshal(data, &connections); err != nil {
		return err
	}

	// Check if we already have connections (migration already done)
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM connections").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // Already migrated
	}

	// Migrate connections
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO connections (id, name, host, port, database, username, password, ssl_mode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, conn := range connections {
		_, err = stmt.Exec(
			conn.ID,
			conn.Name,
			conn.Host,
			conn.Port,
			conn.Database,
			conn.Username,
			conn.Password,
			conn.SSLMode,
		)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	// Backup and remove old file
	backupPath := jsonPath + ".migrated"
	return os.Rename(jsonPath, backupPath)
}
