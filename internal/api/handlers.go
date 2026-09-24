package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/executor"
	"github.com/rudi-asr/ujiscan/internal/models"
	"github.com/rudi-asr/ujiscan/internal/playbook"
	"github.com/rudi-asr/ujiscan/internal/store"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

// Handler holds dependencies for API handlers
type Handler struct {
	scanStore      *store.ScanStore
	executor       *tools.Executor
	scanExec       *executor.ScanExecutor
	playbookEngine *playbook.Engine
}

// NewHandler creates a new API handler
func NewHandler(scanStore *store.ScanStore, toolExecutor *tools.Executor, scanExec *executor.ScanExecutor, pbEngine *playbook.Engine) *Handler {
	return &Handler{
		scanStore:      scanStore,
		executor:       toolExecutor,
		scanExec:       scanExec,
		playbookEngine: pbEngine,
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

// PlaybookRequest represents a playbook scan request
type PlaybookRequest struct {
	Playbook string `json:"playbook"`
	Target   string `json:"target"`
}

// HandlePlaybookScan starts a scan using a playbook
func (h *Handler) HandlePlaybookScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PlaybookRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Try to load playbook - might need to find by name match
	availablePlaybooks, err := h.playbookEngine.ListPlaybooks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load playbooks: %v", err), http.StatusInternalServerError)
		return
	}

	// Find playbook - use explicit mapping to be reliable
	var selectedPlaybook *playbook.Playbook
	searchName := strings.ToLower(strings.TrimSpace(req.Playbook))
	fmt.Printf("[DEBUG] Handler looking for playbook: %s\n", req.Playbook)
	fmt.Printf("[DEBUG] Search name (lowercase): %s\n", searchName)
	fmt.Printf("[DEBUG] Available playbooks: %v\n", len(availablePlaybooks))
	for i, pb := range availablePlaybooks {
		fmt.Printf("[DEBUG]   %d: %s\n", i, pb.Name)
	}
	
	// Direct mapping from request to playbook names
	var targetName string
	switch searchName {
	case "network-discovery", "network discovery":
		targetName = "Network Discovery"
	case "vulnerability-quick", "vulnerability quick scan":
		targetName = "Vulnerability Quick Scan"
	case "web-full-scan", "web server full scan", "web-server-full-scan":
		targetName = "Web Server Full Scan"
	default:
		// Fallback: case-insensitive search
		for _, pb := range availablePlaybooks {
			if strings.ToLower(pb.Name) == searchName {
				targetName = pb.Name
				break
			}
		}
	}
	
	// Find playbook by exact name
	for _, pb := range availablePlaybooks {
		if pb.Name == targetName {
			selectedPlaybook = pb
			break
		}
	}

	if selectedPlaybook == nil {
		http.Error(w, fmt.Sprintf("Playbook %s not found", req.Playbook), http.StatusNotFound)
		return
	}

	// Use playbook filename (without .md extension) for loading
	// Map from API name to filename
	var playbookFilename string
	switch searchName {
	case "network-discovery", "network discovery":
		playbookFilename = "network-discovery"
	case "vulnerability-quick", "vulnerability quick scan":
		playbookFilename = "vulnerability-quick"
	case "web-full-scan", "web full scan", "web server full scan":
		playbookFilename = "web-full-scan"
	default:
		// Try direct filename
		playbookFilename = req.Playbook
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

	// Start playbook execution asynchronously
	go func() {
		fmt.Printf("[handler] Starting playbook execution: %s for scan %s\n", playbookFilename, scan.ID)
		err := h.playbookEngine.ExecutePlaybook(scan.ID, playbookFilename, req.Target)
		fmt.Printf("[handler] Playbook execution completed with error: %v\n", err)
		if err != nil {
			fmt.Printf("[handler] Playbook execution error: %v\n", err)
			// Update scan with error
			h.scanStore.UpdateScanStatus(scan.ID, models.ScanStatusFailed)
		} else {
			// Verify scan has results
			finalScan, _ := h.scanStore.GetScan(scan.ID)
			fmt.Printf("[handler] After execution - scan has %d results\n", len(finalScan.Results))
		}
	}()

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ScanResponse{
		ID:     scan.ID,
		Target: scan.Target,
		Status: string(scan.Status),
	})
}

// HandleListPlaybooks returns all available playbooks
func (h *Handler) HandleListPlaybooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	playbooks, err := h.playbookEngine.ListPlaybooks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load playbooks: %v", err), http.StatusInternalServerError)
		return
	}

	type PlaybookInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Author      string `json:"author"`
		Version     string `json:"version"`
	}

	infos := make([]PlaybookInfo, 0, len(playbooks))
	for _, pb := range playbooks {
		infos = append(infos, PlaybookInfo{
			Name:        pb.Name,
			Description: pb.Description,
			Author:      pb.Author,
			Version:     pb.Version,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}
