package executor

import (
	"fmt"
	"sync"

	"github.com/rudi-asr/ujiscan/internal/models"
	"github.com/rudi-asr/ujiscan/internal/store"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

// ScanExecutor orchestrates scan execution
type ScanExecutor struct {
	store    *store.ScanStore
	executor *tools.Executor
	mu       sync.Mutex
}

// NewScanExecutor creates a new scan executor
func NewScanExecutor(scanStore *store.ScanStore, toolExecutor *tools.Executor) *ScanExecutor {
	return &ScanExecutor{
		store:    scanStore,
		executor: toolExecutor,
	}
}

// ExecuteScanAsync starts a scan asynchronously
func (se *ScanExecutor) ExecuteScanAsync(scanID string) {
	go se.ExecuteScan(scanID)
}

// ExecuteScan runs a scan synchronously
func (se *ScanExecutor) ExecuteScan(scanID string) error {
	// Get scan from store
	scan, err := se.store.GetScan(scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %w", err)
	}

	// Update status to running
	se.store.UpdateScanStatus(scanID, models.ScanStatusRunning)

	// Execute recon phase
	err = se.executePhase(scanID, models.PhaseRecon, scan.Target)
	if err != nil {
		se.store.CompleteScan(scanID, err)
		return err
	}

	// Execute enum phase (if tools available)
	err = se.executePhase(scanID, models.PhaseEnum, scan.Target)
	if err != nil {
		// Don't fail on enum phase - continue
		fmt.Printf("enum phase error: %v\n", err)
	}

	// Mark as completed
	err = se.store.CompleteScan(scanID, nil)
	return err
}

// executePhase runs all tools for a specific phase
func (se *ScanExecutor) executePhase(scanID string, phase models.PhaseType, target string) error {
	switch phase {
	case models.PhaseRecon:
		return se.executeReconPhase(scanID, target)
	case models.PhaseEnum:
		return se.executeEnumPhase(scanID, target)
	default:
		return fmt.Errorf("unknown phase: %s", phase)
	}
}

// executeReconPhase runs nmap quick scan
func (se *ScanExecutor) executeReconPhase(scanID string, target string) error {
	nmap := se.executor.GetToolConfig("nmap")
	if nmap == nil || !nmap.Available {
		return fmt.Errorf("nmap tool not available")
	}

	// Run quick nmap scan
	output, err := se.executor.QuickNmapScan(target)
	if err != nil {
		return fmt.Errorf("nmap scan failed: %w", err)
	}

	// Store result
	err = se.store.AddResult(scanID, *output)
	if err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	return nil
}

// executeEnumPhase runs nuclei scan
func (se *ScanExecutor) executeEnumPhase(scanID string, target string) error {
	nuclei := se.executor.GetToolConfig("nuclei")
	if nuclei == nil || !nuclei.Available {
		return fmt.Errorf("nuclei tool not available")
	}

	// Run nuclei scan
	output, err := se.executor.ScanWithNuclei(target)
	if err != nil {
		return fmt.Errorf("nuclei scan failed: %w", err)
	}

	// Store result
	err = se.store.AddResult(scanID, *output)
	if err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	return nil
}

// ExecuteTool runs a single tool
func (se *ScanExecutor) ExecuteTool(scanID string, toolName string, args []string, phase models.PhaseType, target string) (*models.ToolOutput, error) {
	output, err := se.executor.RunTool(toolName, args, phase, target)
	if err != nil {
		return output, err
	}

	// Store result if scan exists
	if scanID != "" {
		_ = se.store.AddResult(scanID, *output)
	}

	return output, nil
}
