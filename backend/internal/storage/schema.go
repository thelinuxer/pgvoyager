package storage

const schema = `
CREATE TABLE IF NOT EXISTS connections (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	host TEXT NOT NULL,
	port INTEGER NOT NULL,
	database TEXT NOT NULL,
	username TEXT NOT NULL,
	password TEXT NOT NULL,
	ssl_mode TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS query_history (
	id TEXT PRIMARY KEY,
	connection_id TEXT NOT NULL,
	connection_name TEXT NOT NULL,
	sql TEXT NOT NULL,
	duration INTEGER NOT NULL,
	row_count INTEGER NOT NULL,
	success BOOLEAN NOT NULL,
	error TEXT,
	executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_query_history_connection_id ON query_history(connection_id);
CREATE INDEX IF NOT EXISTS idx_query_history_executed_at ON query_history(executed_at DESC);

CREATE TABLE IF NOT EXISTS preferences (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`
