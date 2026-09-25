// Package db provides database pool and optimizations
package db

import (
	"database/sql"
	"fmt"
	"sync"
)

// PoolConfig holds database pool configuration
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // seconds
}

// Pool manages database connections
type Pool struct {
	db     *sql.DB
	mu     sync.RWMutex
	config PoolConfig
}

// NewPool creates a new connection pool
func NewPool(db *sql.DB, config PoolConfig) *Pool {
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 25
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 5
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = 300 // 5 minutes
	}

	pool := &Pool{
		db:     db,
		config: config,
	}

	// Apply settings
	pool.applyConfig()

	return pool
}

// applyConfig applies pool settings to the database
func (p *Pool) applyConfig() {
	p.db.SetMaxOpenConns(p.config.MaxOpenConns)
	p.db.SetMaxIdleConns(p.config.MaxIdleConns)
	p.db.SetConnMaxLifetime(0) // SQLite doesn't use lifetime well
}

// DB returns the underlying database
func (p *Pool) DB() *sql.DB {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
}

// Stats returns connection pool statistics
func (p *Pool) Stats() sql.DBStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db.Stats()
}

// Optimize applies SQLite performance optimizations
func (p *Pool) Optimize() error {
	pragmas := `
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -64000;
		PRAGMA temp_store = MEMORY;
		PRAGMA mmap_size = 30000000;
		PRAGMA page_size = 4096;
		PRAGMA busy_timeout = 5000;
	`

	db := p.DB()

	// Execute pragmas
	_, err := db.Exec(pragmas)
	if err != nil {
		return fmt.Errorf("failed to apply pragmas: %w", err)
	}

	// Run ANALYZE to optimize query planner
	_, err = db.Exec("ANALYZE;")
	if err != nil {
		return fmt.Errorf("failed to analyze: %w", err)
	}

	// Vacuum to optimize database
	_, err = db.Exec("VACUUM;")
	if err != nil {
		return fmt.Errorf("failed to vacuum: %w", err)
	}

	return nil
}

// AddIndexes adds missing indexes for common queries
func (p *Pool) AddIndexes() error {
	db := p.DB()

	indexes := []string{
		// Engagement lookups
		"CREATE INDEX IF NOT EXISTS idx_engagements_status ON engagements(status);",
		"CREATE INDEX IF NOT EXISTS idx_engagements_created_by ON engagements(created_by);",

		// Finding lookups
		"CREATE INDEX IF NOT EXISTS idx_findings_engagement_id ON findings(engagement_id);",
		"CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);",
		"CREATE INDEX IF NOT EXISTS idx_findings_status ON findings(status);",
		"CREATE INDEX IF NOT EXISTS idx_findings_created_by ON findings(created_by);",

		// Engagement team lookups
		"CREATE INDEX IF NOT EXISTS idx_engagement_team_engagement_id ON engagement_team(engagement_id);",
		"CREATE INDEX IF NOT EXISTS idx_engagement_team_user_id ON engagement_team(user_id);",

		// Scope lookups
		"CREATE INDEX IF NOT EXISTS idx_engagement_scope_engagement_id ON engagement_scope(engagement_id);",

		// Comments lookups
		"CREATE INDEX IF NOT EXISTS idx_comments_finding_id ON comments(finding_id);",
		"CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);",

		// Audit log lookups (high volume)
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource);",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);",

		// Notifications lookups
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);",

		// Composite indexes for common queries
		"CREATE INDEX IF NOT EXISTS idx_findings_engagement_severity ON findings(engagement_id, severity);",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_timestamp ON audit_logs(user_id, timestamp);",
	}

	for _, idx := range indexes {
		_, err := db.Exec(idx)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.db.Close()
}
