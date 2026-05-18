package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
	"github.com/thelinuxer/pgvoyager/internal/models"
	"github.com/thelinuxer/pgvoyager/internal/secretstore"
)

var (
	db     *sql.DB
	dbOnce sync.Once
)

// GetDB returns the singleton database instance.
func GetDB() (*sql.DB, error) {
	var err error
	dbOnce.Do(func() {
		var pgvoyagerDir string
		pgvoyagerDir, err = secretstore.Ensure()
		if err != nil {
			return
		}

		dbPath := filepath.Join(pgvoyagerDir, "pgvoyager.db")
		db, err = sql.Open("sqlite", dbPath)
		if err != nil {
			return
		}

		// Tighten file perms — the DB contains plaintext connection
		// passwords. SQLite respects the user's umask on file create,
		// which is typically 0644 (world-readable). Force 0600 so
		// other accounts on the host can't read it.
		if err = secretstore.SecureFile(dbPath); err != nil {
			return
		}

		if _, err = db.Exec(schema); err != nil {
			return
		}

		err = migrateFromJSON(pgvoyagerDir)
	})
	return db, err
}

// migrateFromJSON migrates data from the legacy connections.json file. After
// a successful import the backup is shredded — the prior implementation
// renamed it to `.migrated`, leaving plaintext passwords on disk forever
// at whatever perms the user had originally chosen.
func migrateFromJSON(configDir string) error {
	jsonPath := filepath.Join(configDir, "connections.json")
	data, err := os.ReadFile(jsonPath)
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

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM connections").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		// Already migrated on a prior run — still scrub the leftover
		// JSON (or any prior `.migrated` rename) so plaintext doesn't
		// linger.
		_ = secretstore.ShredAndRemove(jsonPath)
		_ = secretstore.ShredAndRemove(jsonPath + ".migrated")
		return nil
	}

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

	// Successful import — destroy the plaintext source.
	if err := secretstore.ShredAndRemove(jsonPath); err != nil {
		return fmt.Errorf("shred migrated json: %w", err)
	}
	// Also nuke any prior `.migrated` backup left by older installs.
	_ = secretstore.ShredAndRemove(jsonPath + ".migrated")
	return nil
}
