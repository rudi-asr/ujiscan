// Package db provides database initialization and schema
package db

import (
	"database/sql"
	"fmt"
)

// InitDB initializes the SQLite database and creates tables
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// createTables creates all required database tables
func createTables(db *sql.DB) error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Engagements table
	CREATE TABLE IF NOT EXISTS engagements (
		id TEXT PRIMARY KEY,
		client_name TEXT NOT NULL,
		project_name TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		start_date DATETIME,
		end_date DATETIME,
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (created_by) REFERENCES users(id)
	);

	-- Engagement scope (targets)
	CREATE TABLE IF NOT EXISTS engagement_scope (
		id TEXT PRIMARY KEY,
		engagement_id TEXT NOT NULL,
		target TEXT NOT NULL,
		FOREIGN KEY (engagement_id) REFERENCES engagements(id)
	);

	-- Engagement team (assigned pentesters)
	CREATE TABLE IF NOT EXISTS engagement_team (
		id TEXT PRIMARY KEY,
		engagement_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT,
		FOREIGN KEY (engagement_id) REFERENCES engagements(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Findings table
	CREATE TABLE IF NOT EXISTS findings (
		id TEXT PRIMARY KEY,
		engagement_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		severity TEXT,
		cvss_score REAL,
		status TEXT NOT NULL DEFAULT 'open',
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (engagement_id) REFERENCES engagements(id),
		FOREIGN KEY (created_by) REFERENCES users(id)
	);

	-- Comments table
	CREATE TABLE IF NOT EXISTS comments (
		id TEXT PRIMARY KEY,
		finding_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (finding_id) REFERENCES findings(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Audit logs table
	CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		resource_id TEXT,
		changes TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Notifications table
	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT,
		read INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Data snapshots (for serialized persistence)
	CREATE TABLE IF NOT EXISTS data_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		data BLOB NOT NULL,
		saved_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_snapshots_key ON data_snapshots(key);
	`

	_, err := db.Exec(schema)
	return err
}
