package storage

import (
	"database/sql"
	"time"
)

// QueryHistoryEntry represents a single query execution record
type QueryHistoryEntry struct {
	ID             string    `json:"id"`
	ConnectionID   string    `json:"connectionId"`
	ConnectionName string    `json:"connectionName"`
	SQL            string    `json:"sql"`
	Duration       int64     `json:"duration"`
	RowCount       int       `json:"rowCount"`
	Success        bool      `json:"success"`
	Error          string    `json:"error,omitempty"`
	ExecutedAt     time.Time `json:"executedAt"`
}

const maxHistoryEntries = 100

// AddQueryHistory adds a query execution to the history
func AddQueryHistory(entry *QueryHistoryEntry) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO query_history (id, connection_id, connection_name, sql, duration, row_count, success, error, executed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.ConnectionID, entry.ConnectionName, entry.SQL, entry.Duration, entry.RowCount, entry.Success, entry.Error, entry.ExecutedAt)
	if err != nil {
		return err
	}

	// Clean up old entries beyond max limit
	return cleanOldHistory(db)
}

// GetQueryHistory retrieves query history with optional filtering
func GetQueryHistory(connectionID string, limit int) ([]QueryHistoryEntry, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > maxHistoryEntries {
		limit = maxHistoryEntries
	}

	var rows *sql.Rows
	if connectionID != "" {
		rows, err = db.Query(`
			SELECT id, connection_id, connection_name, sql, duration, row_count, success, error, executed_at
			FROM query_history
			WHERE connection_id = ?
			ORDER BY executed_at DESC
			LIMIT ?
		`, connectionID, limit)
	} else {
		rows, err = db.Query(`
			SELECT id, connection_id, connection_name, sql, duration, row_count, success, error, executed_at
			FROM query_history
			ORDER BY executed_at DESC
			LIMIT ?
		`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []QueryHistoryEntry{}
	for rows.Next() {
		var entry QueryHistoryEntry
		var errorStr sql.NullString
		err := rows.Scan(
			&entry.ID,
			&entry.ConnectionID,
			&entry.ConnectionName,
			&entry.SQL,
			&entry.Duration,
			&entry.RowCount,
			&entry.Success,
			&errorStr,
			&entry.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}
		if errorStr.Valid {
			entry.Error = errorStr.String
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// DeleteQueryHistory removes a specific query history entry
func DeleteQueryHistory(id string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM query_history WHERE id = ?", id)
	return err
}

// ClearQueryHistory removes all query history or history for a specific connection
func ClearQueryHistory(connectionID string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	if connectionID != "" {
		_, err = db.Exec("DELETE FROM query_history WHERE connection_id = ?", connectionID)
	} else {
		_, err = db.Exec("DELETE FROM query_history")
	}
	return err
}

func cleanOldHistory(db *sql.DB) error {
	_, err := db.Exec(`
		DELETE FROM query_history
		WHERE id NOT IN (
			SELECT id FROM query_history
			ORDER BY executed_at DESC
			LIMIT ?
		)
	`, maxHistoryEntries)
	return err
}
