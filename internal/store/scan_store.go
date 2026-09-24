package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rudi-asr/ujiscan/internal/models"
)

// ScanStore manages in-memory scan storage
type ScanStore struct {
	scans map[string]*models.Scan
	mu    sync.RWMutex
}

// NewScanStore creates a new scan store
func NewScanStore() *ScanStore {
	return &ScanStore{
		scans: make(map[string]*models.Scan),
	}
}

// CreateScan creates a new scan and stores it
func (s *ScanStore) CreateScan(target string) (*models.Scan, error) {
	if target == "" {
		return nil, fmt.Errorf("target cannot be empty")
	}

	scan := &models.Scan{
		ID:        uuid.New().String(),
		Target:    target,
		Status:    models.ScanStatusPending,
		StartedAt: time.Now(),
		Results:   make([]models.ToolOutput, 0),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.scans[scan.ID] = scan
	return scan, nil
}

// GetScan retrieves a scan by ID
func (s *ScanStore) GetScan(id string) (*models.Scan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scan, ok := s.scans[id]
	if !ok {
		return nil, fmt.Errorf("scan %s not found", id)
	}

	return scan, nil
}

// UpdateScanStatus updates the status of a scan
func (s *ScanStore) UpdateScanStatus(id string, status models.ScanStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.Status = status
	return nil
}

// AddResult adds a tool output to scan results
func (s *ScanStore) AddResult(id string, result models.ToolOutput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.Results = append(scan.Results, result)
	return nil
}

// CompleteScan marks a scan as completed
func (s *ScanStore) CompleteScan(id string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	now := time.Now()
	scan.EndedAt = &now
	scan.Status = models.ScanStatusCompleted

	if err != nil {
		scan.Status = models.ScanStatusFailed
		scan.Error = err.Error()
	}

	return nil
}

// ListScans returns all scans (limited to last N)
func (s *ScanStore) ListScans(limit int) []*models.Scan {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scans := make([]*models.Scan, 0, len(s.scans))
	for _, scan := range s.scans {
		scans = append(scans, scan)
	}

	// Sort by started time descending (newest first)
	// For now, just return all (proper sort later)
	if limit > 0 && len(scans) > limit {
		return scans[:limit]
	}

	return scans
}

// DeleteScan removes a scan from storage
func (s *ScanStore) DeleteScan(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.scans[id]; !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	delete(s.scans, id)
	return nil
}

// ClearScans deletes all scans (for testing)
func (s *ScanStore) ClearScans() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scans = make(map[string]*models.Scan)
}

// GetStats returns statistics about stored scans
func (s *ScanStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total":     len(s.scans),
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}

	for _, scan := range s.scans {
		switch scan.Status {
		case models.ScanStatusPending:
			stats["pending"] = stats["pending"].(int) + 1
		case models.ScanStatusRunning:
			stats["running"] = stats["running"].(int) + 1
		case models.ScanStatusCompleted:
			stats["completed"] = stats["completed"].(int) + 1
		case models.ScanStatusFailed:
			stats["failed"] = stats["failed"].(int) + 1
		}
	}

	return stats
}
