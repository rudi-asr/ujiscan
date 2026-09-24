package store

import (
	"fmt"
	"testing"

	"github.com/rudi-asr/ujiscan/internal/models"
)

func TestCreateScan(t *testing.T) {
	store := NewScanStore()

	scan, err := store.CreateScan("192.168.1.1")
	if err != nil {
		t.Fatalf("CreateScan failed: %v", err)
	}

	if scan.ID == "" {
		t.Fatal("Scan ID is empty")
	}

	if scan.Target != "192.168.1.1" {
		t.Fatalf("Expected target 192.168.1.1, got %s", scan.Target)
	}

	if scan.Status != models.ScanStatusPending {
		t.Fatalf("Expected status pending, got %s", scan.Status)
	}
}

func TestGetScan(t *testing.T) {
	store := NewScanStore()

	scan1, _ := store.CreateScan("192.168.1.1")
	_, _ = store.CreateScan("example.com")

	// Get existing scan
	retrieved, err := store.GetScan(scan1.ID)
	if err != nil {
		t.Fatalf("GetScan failed: %v", err)
	}

	if retrieved.Target != "192.168.1.1" {
		t.Fatalf("Expected target 192.168.1.1, got %s", retrieved.Target)
	}

	// Get non-existent scan
	_, err = store.GetScan("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent scan")
	}
}

func TestUpdateScanStatus(t *testing.T) {
	store := NewScanStore()
	scan, _ := store.CreateScan("192.168.1.1")

	err := store.UpdateScanStatus(scan.ID, models.ScanStatusRunning)
	if err != nil {
		t.Fatalf("UpdateScanStatus failed: %v", err)
	}

	retrieved, _ := store.GetScan(scan.ID)
	if retrieved.Status != models.ScanStatusRunning {
		t.Fatalf("Expected status running, got %s", retrieved.Status)
	}
}

func TestAddResult(t *testing.T) {
	store := NewScanStore()
	scan, _ := store.CreateScan("192.168.1.1")

	result := models.ToolOutput{
		ID:       "test-result",
		ToolName: "nmap",
		Target:   "192.168.1.1",
		Success:  true,
	}

	err := store.AddResult(scan.ID, result)
	if err != nil {
		t.Fatalf("AddResult failed: %v", err)
	}

	retrieved, _ := store.GetScan(scan.ID)
	if len(retrieved.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(retrieved.Results))
	}

	if retrieved.Results[0].ToolName != "nmap" {
		t.Fatalf("Expected tool nmap, got %s", retrieved.Results[0].ToolName)
	}
}

func TestCompleteScan(t *testing.T) {
	store := NewScanStore()
	scan, _ := store.CreateScan("192.168.1.1")

	err := store.CompleteScan(scan.ID, nil)
	if err != nil {
		t.Fatalf("CompleteScan failed: %v", err)
	}

	retrieved, _ := store.GetScan(scan.ID)
	if retrieved.Status != models.ScanStatusCompleted {
		t.Fatalf("Expected status completed, got %s", retrieved.Status)
	}

	if retrieved.EndedAt == nil {
		t.Fatal("EndedAt is nil")
	}
}

func TestCompleteScanWithError(t *testing.T) {
	store := NewScanStore()
	scan, _ := store.CreateScan("192.168.1.1")

	testErr := "test error"
	err := store.CompleteScan(scan.ID, fmt.Errorf("%s", testErr))
	if err != nil {
		t.Fatalf("CompleteScan failed: %v", err)
	}

	retrieved, _ := store.GetScan(scan.ID)
	if retrieved.Status != models.ScanStatusFailed {
		t.Fatalf("Expected status failed, got %s", retrieved.Status)
	}

	if retrieved.Error == "" {
		t.Fatal("Error message is empty")
	}
}

func TestListScans(t *testing.T) {
	store := NewScanStore()

	for i := 0; i < 5; i++ {
		store.CreateScan("target-" + fmt.Sprintf("%d", i))
	}

	scans := store.ListScans(0)
	if len(scans) != 5 {
		t.Fatalf("Expected 5 scans, got %d", len(scans))
	}

	scans = store.ListScans(2)
	if len(scans) != 2 {
		t.Fatalf("Expected 2 scans with limit, got %d", len(scans))
	}
}

func TestGetStats(t *testing.T) {
	store := NewScanStore()

	scan1, _ := store.CreateScan("192.168.1.1")
	store.UpdateScanStatus(scan1.ID, models.ScanStatusRunning)

	scan2, _ := store.CreateScan("example.com")
	store.CompleteScan(scan2.ID, nil)

	stats := store.GetStats()
	if stats["total"] != 2 {
		t.Fatalf("Expected total 2, got %d", stats["total"])
	}

	if stats["running"] != 1 {
		t.Fatalf("Expected running 1, got %d", stats["running"])
	}

	if stats["completed"] != 1 {
		t.Fatalf("Expected completed 1, got %d", stats["completed"])
	}
}

func TestDeleteScan(t *testing.T) {
	store := NewScanStore()
	scan, _ := store.CreateScan("192.168.1.1")

	err := store.DeleteScan(scan.ID)
	if err != nil {
		t.Fatalf("DeleteScan failed: %v", err)
	}

	_, err = store.GetScan(scan.ID)
	if err == nil {
		t.Fatal("Expected error for deleted scan")
	}
}
