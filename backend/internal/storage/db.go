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

		// Open + initialize the DB under a 0077 umask so the SQLite
		// file (and its WAL / journal sidecars created later by libc)
		// are born 0600. Post-creation chmod was the prior approach
		// and it tripped SQLITE_READONLY_DBMOVED on modernc.org/sqlite
		// — the library noticed the inode mode change between the
		// initial Open and the first write and refused subsequent
		// writes.
		err = secretstore.WithSecretUmask(func() error {
			var openErr error
			db, openErr = sql.Open("sqlite", dbPath)
			if openErr != nil {
				return openErr
			}
			if _, e := db.Exec(schema); e != nil {
				return e
			}
			return migrateFromJSON(pgvoyagerDir)
		})

		// Defensive belt-and-suspenders: if the file already existed
		// from a pre-umask install at 0644, tighten it now. Skipping
		// the chmod when perms are already <= 0600 avoids tripping
		// the same DBMOVED detection on a fresh DB.
		if err == nil {
			_ = tightenIfWorldReadable(dbPath)
		}
	})
	return db, err
}

// tightenIfWorldReadable lowers the DB file to 0600 only when it's
// currently more permissive. No-op on already-tight files so we don't
// chmod the file on every cold start (which could race with libsqlite's
// inode tracking).
func tightenIfWorldReadable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm() <= secretstore.FilePerm {
		return nil
	}
	return secretstore.SecureFile(path)
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
