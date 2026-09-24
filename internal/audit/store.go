// Package audit provides in-memory storage
package audit

import (
	"sort"
	"sync"
	"time"
)

// MemoryAuditStore implements AuditStore using in-memory storage
type MemoryAuditStore struct {
	mu    sync.RWMutex
	logs  []*AuditLog
	index map[string]*AuditLog
}

// NewMemoryAuditStore creates a new in-memory audit store
func NewMemoryAuditStore() *MemoryAuditStore {
	return &MemoryAuditStore{
		logs:  make([]*AuditLog, 0),
		index: make(map[string]*AuditLog),
	}
}

// Log stores an audit log entry
func (mas *MemoryAuditStore) Log(log *AuditLog) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	mas.logs = append(mas.logs, log)
	mas.index[log.ID] = log

	return nil
}

// GetLog retrieves a specific audit log
func (mas *MemoryAuditStore) GetLog(id string) (*AuditLog, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	log, exists := mas.index[id]
	if !exists {
		return nil, ErrAuditLogNotFound
	}

	return log, nil
}

// ListLogs retrieves all audit logs (newest first)
func (mas *MemoryAuditStore) ListLogs() ([]*AuditLog, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	result := make([]*AuditLog, len(mas.logs))
	copy(result, mas.logs)

	// Sort by timestamp (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

// ListLogsByUser retrieves logs for a specific user
func (mas *MemoryAuditStore) ListLogsByUser(userID string) ([]*AuditLog, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	var result []*AuditLog
	for _, log := range mas.logs {
		if log.UserID == userID {
			result = append(result, log)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

// ListLogsByAction retrieves logs for a specific action
func (mas *MemoryAuditStore) ListLogsByAction(action Action) ([]*AuditLog, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	var result []*AuditLog
	for _, log := range mas.logs {
		if log.Action == action {
			result = append(result, log)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

// ListLogsByDateRange retrieves logs within a date range
func (mas *MemoryAuditStore) ListLogsByDateRange(startTime, endTime time.Time) ([]*AuditLog, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	var result []*AuditLog
	for _, log := range mas.logs {
		if !log.Timestamp.Before(startTime) && !log.Timestamp.After(endTime) {
			result = append(result, log)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

// ListLogsByResource retrieves logs for a specific resource
func (mas *MemoryAuditStore) ListLogsByResource(resource string) ([]*AuditLog, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	var result []*AuditLog
	for _, log := range mas.logs {
		if log.Resource == resource {
			result = append(result, log)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

// DeleteOldLogs removes logs older than specified time
func (mas *MemoryAuditStore) DeleteOldLogs(olderThan time.Time) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	var newLogs []*AuditLog
	for _, log := range mas.logs {
		if log.Timestamp.After(olderThan) {
			newLogs = append(newLogs, log)
			continue
		}
		// Remove from index
		delete(mas.index, log.ID)
	}

	mas.logs = newLogs
	return nil
}
