// Package persistence provides data persistence utilities
package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// DataSnapshot represents a complete snapshot of application data
type DataSnapshot struct {
	Engagements json.RawMessage `json:"engagements"`
	Findings    json.RawMessage `json:"findings"`
	Comments    json.RawMessage `json:"comments"`
	AuditLogs   json.RawMessage `json:"audit_logs"`
	SavedAt     time.Time       `json:"saved_at"`
}

// PersistenceManager manages data persistence to SQLite
type PersistenceManager struct {
	db *sql.DB
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(db *sql.DB) *PersistenceManager {
	return &PersistenceManager{db: db}
}

// SaveSnapshot saves a data snapshot
func (pm *PersistenceManager) SaveSnapshot(key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO data_snapshots (key, data, saved_at)
		VALUES (?, ?, ?)
	`

	_, err = pm.db.Exec(query, key, jsonData, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	log.Printf("✅ Saved %s snapshot (%d bytes)", key, len(jsonData))
	return nil
}

// LoadSnapshot loads a data snapshot
func (pm *PersistenceManager) LoadSnapshot(key string, dest interface{}) error {
	query := `SELECT data FROM data_snapshots WHERE key = ? ORDER BY saved_at DESC LIMIT 1`

	var jsonData []byte
	err := pm.db.QueryRow(query, key).Scan(&jsonData)
	if err == sql.ErrNoRows {
		return nil // No data saved yet
	}
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	err = json.Unmarshal(jsonData, dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	log.Printf("✅ Loaded %s snapshot (%d bytes)", key, len(jsonData))
	return nil
}

// CreateTable creates the data_snapshots table
func CreatePersistenceTable(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS data_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL,
		data BLOB NOT NULL,
		saved_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(key)
	);
	CREATE INDEX IF NOT EXISTS idx_snapshots_key ON data_snapshots(key);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create persistence table: %w", err)
	}

	return nil
}
