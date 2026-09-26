package store

import (
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// ScanStoreInterface — kontrak store scan yang dipakai handler & playbook engine.
// Mendukung implementasi in-memory (ScanStore) dan persistent (ScanFileStore).
type ScanStoreInterface interface {
	CreateScan(target string) (*models.Scan, error)
	GetScan(id string) (*models.Scan, error)
	UpdateScanStatus(id string, status models.ScanStatus) error
	AddResult(id string, result models.ToolOutput) error
	AddFinding(id string, finding models.Finding) error
	AddAIDecision(id string, decision models.AIDecision) error
	SetAIReport(id string, report string) error
	SetScanMeta(id string, scanType string, model string) error
	CompleteScan(id string, err error) error
	DeleteScan(id string) error
	GetStats() map[string]interface{}
	ListScans(limit int) []*models.Scan
}

// ScanStats — struktur statistik umum (dipakai HandleStats).
type ScanStats struct {
	TotalScans   int       `json:"total_scans"`
	Running      int       `json:"running"`
	Completed    int       `json:"completed"`
	Failed       int       `json:"failed"`
	Pending      int       `json:"pending"`
	LastScanTime time.Time `json:"last_scan_time,omitempty"`
}

var _ ScanStoreInterface = (*ScanStore)(nil)
var _ ScanStoreInterface = (*ScanFileStore)(nil)