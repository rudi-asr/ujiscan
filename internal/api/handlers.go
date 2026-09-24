package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/executor"
	"github.com/rudi-asr/ujiscan/internal/store"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

// Handler holds dependencies for API handlers
type Handler struct {
	scanStore  *store.ScanStore
	executor   *tools.Executor
	scanExec   *executor.ScanExecutor
}

// NewHandler creates a new API handler
func NewHandler(scanStore *store.ScanStore, toolExecutor *tools.Executor, scanExec *executor.ScanExecutor) *Handler {
	return &Handler{
		scanStore: scanStore,
		executor:  toolExecutor,
		scanExec:  scanExec,
	}
}

// ToolInfo represents tool information in JSON
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	BinaryPath  string `json:"binary_path"`
}

// HandleListTools returns all available tools
func (h *Handler) HandleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := h.executor.ListTools()
	toolInfos := make([]ToolInfo, 0, len(tools))

	for _, tool := range tools {
		toolInfos = append(toolInfos, ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Available:   tool.Available,
			BinaryPath:  tool.BinaryPath,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolInfos)
}

// ScanRequest represents a scan request
type ScanRequest struct {
	Target string `json:"target"`
}

// ScanResponse represents a scan response
type ScanResponse struct {
	ID     string `json:"id"`
	Target string `json:"target"`
	Status string `json:"status"`
}

// HandleStartScan starts a new scan
func (h *Handler) HandleStartScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Target) == "" {
		http.Error(w, "Target cannot be empty", http.StatusBadRequest)
		return
	}

	// Create scan
	scan, err := h.scanStore.CreateScan(req.Target)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create scan: %v", err), http.StatusInternalServerError)
		return
	}

	// Start scan asynchronously
	h.scanExec.ExecuteScanAsync(scan.ID)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ScanResponse{
		ID:     scan.ID,
		Target: scan.Target,
		Status: string(scan.Status),
	})
}

// HandleGetScan returns scan details and results
func (h *Handler) HandleGetScan(w http.ResponseWriter, r *http.Request, scanID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scan, err := h.scanStore.GetScan(scanID)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scan)
}

// HandleDeleteScan deletes a scan
func (h *Handler) HandleDeleteScan(w http.ResponseWriter, r *http.Request, scanID string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.scanStore.DeleteScan(scanID)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"deleted","id":"%s"}`, scanID)
}

// HandleStats returns scan statistics
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.scanStore.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
