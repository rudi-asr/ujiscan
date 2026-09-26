package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// ScanFileStore — ScanStore dengan persistence ke file JSON.
// Data scan tersimpan di <dataDir>/scans.json; aman dari restart container.
// Format: map[scanID]Scan (serializable).
type ScanFileStore struct {
	mu      sync.RWMutex
	scans   map[string]*models.Scan
	dataDir string
	file    string
}

// NewScanFileStore membuat store dengan persistence.
// dataDir: folder penyimpanan (default "/app/data" atau "./data").
func NewScanFileStore(dataDir string) *ScanFileStore {
	if dataDir == "" {
		dataDir = "data"
	}
	s := &ScanFileStore{
		scans:   make(map[string]*models.Scan),
		dataDir: dataDir,
		file:    filepath.Join(dataDir, "scans.json"),
	}
	s.load()
	return s
}

// fileStoreEntry — struktur serialisasi (models.Scan sudah JSON-ready).
type fileStoreEntry struct {
	Scans map[string]*models.Scan `json:"scans"`
}

// load membaca scans.json jika ada (saat startup / recovery).
func (s *ScanFileStore) load() {
	raw, err := os.ReadFile(s.file)
	if err != nil {
		return // belum ada file — mulai kosong
	}
	var entry fileStoreEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		fmt.Printf("[store] Gagal load scans.json: %v\n", err)
		return
	}
	if entry.Scans != nil {
		s.scans = entry.Scans
	}
	fmt.Printf("[store] Loaded %d scan(s) dari %s\n", len(s.scans), s.file)
}

// persist menulis seluruh scan ke disk (di bawah lock).
func (s *ScanFileStore) persist() {
	_ = os.MkdirAll(s.dataDir, 0o755)
	entry := fileStoreEntry{Scans: s.scans}
	raw, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		fmt.Printf("[store] Gagal marshal scans: %v\n", err)
		return
	}
	if err := os.WriteFile(s.file, raw, 0o644); err != nil {
		fmt.Printf("[store] Gagal tulis scans.json: %v\n", err)
	}
}

// CreateScan creates a new scan and stores it (persisted).
func (s *ScanFileStore) CreateScan(target string) (*models.Scan, error) {
	if target == "" {
		return nil, fmt.Errorf("target cannot be empty")
	}

	scan := &models.Scan{
		ID:        newID(),
		Target:    target,
		Status:    models.ScanStatusPending,
		StartedAt: time.Now(),
		Results:   make([]models.ToolOutput, 0),
		Findings:  make([]models.Finding, 0),
	}

	s.mu.Lock()
	s.scans[scan.ID] = scan
	s.persist()
	s.mu.Unlock()
	return scan, nil
}

// GetScan retrieves a scan by ID.
func (s *ScanFileStore) GetScan(id string) (*models.Scan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scan, ok := s.scans[id]
	if !ok {
		return nil, fmt.Errorf("scan %s not found", id)
	}

	return scan, nil
}

// UpdateScanStatus updates the status of a scan (persisted).
func (s *ScanFileStore) UpdateScanStatus(id string, status models.ScanStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.Status = status
	s.persist()
	return nil
}

// AddResult appends a tool result (persisted).
func (s *ScanFileStore) AddResult(id string, result models.ToolOutput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.Results = append(scan.Results, result)
	s.persist()
	return nil
}

// AddFinding adds a finding to a scan (persisted).
func (s *ScanFileStore) AddFinding(id string, finding models.Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.Findings = append(scan.Findings, finding)
	s.persist()
	return nil
}

// AddAIDecision appends an AI decision (per phase) ke scan (persisted).
func (s *ScanFileStore) AddAIDecision(id string, decision models.AIDecision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.AIDecisions = append(scan.AIDecisions, decision)
	scan.AIReasoning = decision.Analysis
	s.persist()
	return nil
}

// SetAIReport stores the final AI-generated report (markdown) ke scan (persisted).
func (s *ScanFileStore) SetAIReport(id string, report string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.AIReport = report
	s.persist()
	return nil
}

// SetScanMeta stores scan type + AI model (persisted).
func (s *ScanFileStore) SetScanMeta(id string, scanType string, model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scan, ok := s.scans[id]
	if !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	scan.ScanType = scanType
	scan.Model = model
	s.persist()
	return nil
}

// CompleteScan marks a scan as completed (persisted).
func (s *ScanFileStore) CompleteScan(id string, err error) error {
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

	s.persist()
	return nil
}

// ListScans returns all scans (terbaru dulu) — untuk history dashboard.
func (s *ScanFileStore) ListScans(limit int) []*models.Scan {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]*models.Scan, 0, len(s.scans))
	for _, sc := range s.scans {
		all = append(all, sc)
	}

	// Urutkan: terbaru duluan (by StartedAt desc)
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].StartedAt.After(all[j-1].StartedAt); j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// newID generates a unique ID (UUID v4).
func newID() string {
	return fmt.Sprintf("scan_%d", time.Now().UnixNano())
}

// DeleteScan removes a scan (persisted).
func (s *ScanFileStore) DeleteScan(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.scans[id]; !ok {
		return fmt.Errorf("scan %s not found", id)
	}

	delete(s.scans, id)
	s.persist()
	return nil
}

// GetStats returns scan statistics.
func (s *ScanFileStore) GetStats() map[string]interface{} {
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