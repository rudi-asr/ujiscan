package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/models"
	"github.com/rudi-asr/ujiscan/internal/playbook"
	"github.com/rudi-asr/ujiscan/internal/reports"
	"github.com/rudi-asr/ujiscan/internal/store"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

// Handler holds dependencies for API handlers
type Handler struct {
	scanStore      store.ScanStoreInterface
	executor       *tools.Executor
	playbookEngine *playbook.Engine
}

// NewHandler creates a new API handler
func NewHandler(scanStore store.ScanStoreInterface, toolExecutor *tools.Executor, pbEngine *playbook.Engine) *Handler {
	return &Handler{
		scanStore:      scanStore,
		executor:       toolExecutor,
		playbookEngine: pbEngine,
	}
}

// HandleListScans returns scan history (terbaru dulu) — untuk dashboard.
func (h *Handler) HandleListScans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 20
	scans := h.scanStore.ListScans(limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
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

	// Return default hostedscan tools available in playbooks
	toolInfos := []ToolInfo{
		{Name: "dig", Description: "DNS lookup and enumeration", Available: true},
		{Name: "subfinder", Description: "Subdomain enumeration", Available: true},
		{Name: "nmap", Description: "Network mapping and port scanning", Available: true},
		{Name: "httpx", Description: "HTTP probing and fingerprinting", Available: true},
		{Name: "whatweb", Description: "Web technology identification", Available: true},
		{Name: "sslscan", Description: "SSL/TLS security audit", Available: true},
		{Name: "nuclei", Description: "Vulnerability scanning with templates", Available: true},
		{Name: "gobuster", Description: "Directory and DNS brute force", Available: true},
		{Name: "nikto", Description: "Web server vulnerability scanner", Available: true},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolInfos)
}

// ScanRequest represents a scan request
type ScanRequest struct {
	Target    string   `json:"target"`
	ScanType  string   `json:"scanType"`  // "regular", "quick", "full-scan-ai"
	Tools     []string `json:"tools"`     // For quick scan: user-selected tools subset
	Model     string   `json:"model"`     // For full-scan-ai: "deepseek", "openai", "claude"
	Objective string   `json:"objective"` // For full-scan-ai: AI instruction/objective
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

	// Default to regular scan if scanType not specified
	if strings.TrimSpace(req.ScanType) == "" {
		req.ScanType = "regular"
	}

	// Create scan
	scan, err := h.scanStore.CreateScan(req.Target)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create scan: %v", err), http.StatusInternalServerError)
		return
	}

	// Store scan type + model meta untuk UI
	h.scanStore.SetScanMeta(scan.ID, req.ScanType, req.Model)

	// Route to appropriate executor based on scanType
	go func() {
		ctx := context.Background()
		switch req.ScanType {
		case "quick":
			// Quick scan: user-selected tools from regular scan
			if len(req.Tools) > 0 {
				// Execute quick scan with user-selected tools
				err := h.playbookEngine.ExecuteQuickScanWithTools(ctx, scan.ID, req.Target, req.Tools)
				if err != nil {
					h.scanStore.CompleteScan(scan.ID, err)
				}
			} else {
				// No tools selected, use default quick scan (nmap only)
				err := h.playbookEngine.ExecutePlaybook(scan.ID, "quick-scan", req.Target)
				if err != nil {
					h.scanStore.CompleteScan(scan.ID, err)
				}
			}
		case "full-scan-ai":
			// Full scan with AI orchestration (provider: deepseek/openai/claude)
			provider := strings.TrimSpace(req.Model)
			if provider == "" {
				provider = "deepseek" // default
			}
			executor := playbook.NewAgenticExecutor(h.playbookEngine, provider)
			objective := strings.TrimSpace(req.Objective)
			if objective == "" {
				objective = "comprehensive"
			}
			err := executor.ExecuteAgenticScan(ctx, scan.ID, req.Target, objective)
			if err != nil {
				h.scanStore.CompleteScan(scan.ID, err)
			}
		default:
			// Regular scan: all tools from regular-scan playbook
			err := h.playbookEngine.ExecutePlaybook(scan.ID, "regular-scan", req.Target)
			if err != nil {
				h.scanStore.CompleteScan(scan.ID, err)
			}
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
	Playbook      string   `json:"playbook"`
	Target        string   `json:"target"`
	SelectedTools []string `json:"selectedTools"` // Tools selected for quick scan
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

	// Map request name to playbook filename
	searchName := strings.ToLower(strings.TrimSpace(req.Playbook))
	var playbookFilename string

	// Direct mapping from request to filename
	switch searchName {
	case "network-discovery", "network discovery":
		playbookFilename = "network-discovery"
	case "vulnerability-quick", "vulnerability quick scan":
		playbookFilename = "vulnerability-quick"
	case "web-full-scan", "web server full scan", "web-server-full-scan":
		playbookFilename = "web-full-scan"
	case "quick-scan":
		playbookFilename = "quick-scan"
	case "regular-scan":
		playbookFilename = "regular-scan"
	case "full-scan-ai":
		playbookFilename = "full-scan-ai"
	default:
		// Handle dynamic quick-scan patterns
		if strings.HasPrefix(searchName, "quick-scan-") {
			// Use regular-scan playbook for dynamic tool selection
			playbookFilename = "regular-scan"
		} else {
			playbookFilename = req.Playbook
		}
	}

	// Map request name to playbook filename is done above via switch.
	// No need to verify - if file doesn't exist, loader will error during execution.

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

// HandleAgenticPlaybookScan starts an agentic scan using a playbook with AI decision-making
func (h *Handler) HandleAgenticPlaybookScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body with additional objective field
	var req struct {
		Playbook  string `json:"playbook"`
		Target    string `json:"target"`
		Objective string `json:"objective"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Map request name to playbook filename
	searchName := strings.ToLower(strings.TrimSpace(req.Playbook))
	var playbookFilename string

	// Direct mapping from request to filename
	switch searchName {
	case "network-discovery", "network discovery":
		playbookFilename = "network-discovery"
	case "vulnerability-quick", "vulnerability quick scan":
		playbookFilename = "vulnerability-quick"
	case "web-full-scan", "web server full scan", "web-server-full-scan":
		playbookFilename = "web-full-scan"
	default:
		playbookFilename = req.Playbook
	}

	if strings.TrimSpace(req.Target) == "" {
		http.Error(w, "Target cannot be empty", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Objective) == "" {
		// Default objective if not provided
		req.Objective = fmt.Sprintf("Comprehensive penetration test of %s", req.Target)
	}

	// Create scan
	scan, err := h.scanStore.CreateScan(req.Target)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create scan: %v", err), http.StatusInternalServerError)
		return
	}

	// Start agentic playbook execution asynchronously
	go func() {
		fmt.Printf("[handler] Starting AGENTIC playbook execution: %s for scan %s (objective: %s)\n", playbookFilename, scan.ID, req.Objective)
		err := h.playbookEngine.ExecuteAgenticPlaybook(scan.ID, playbookFilename, req.Target, req.Objective)
		fmt.Printf("[handler] Agentic playbook execution completed with error: %v\n", err)
		if err != nil {
			fmt.Printf("[handler] Agentic playbook execution error: %v\n", err)
			// Update scan with error
			h.scanStore.UpdateScanStatus(scan.ID, models.ScanStatusFailed)
		} else {
			// Verify scan has results
			finalScan, _ := h.scanStore.GetScan(scan.ID)
			fmt.Printf("[handler] After agentic execution - scan has %d results\n", len(finalScan.Results))
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

// HandleGenerateReport generates a report for a scan
func (h *Handler) HandleGenerateReport(w http.ResponseWriter, r *http.Request, scanID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get scan
	scan, err := h.scanStore.GetScan(scanID)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	// Parse request for report options
	var req struct {
		ClientName string `json:"client_name"`
		Format     string `json:"format"` // json, html, markdown
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.ClientName == "" {
		req.ClientName = "Assessment Client"
	}
	if req.Format == "" {
		req.Format = "json"
	}

	// Convert scan results to findings
	findings := h.scanResultsToFindings(scan.Results)

	// Generate report
	gen := reports.NewGenerator()
	report := gen.GenerateReport(
		req.ClientName,
		scan.Target,
		[]string{scan.Target},
		"security-assessment",
		scanID,
		findings,
		0, // duration (can be enhanced)
		scan.StartedAt,
	)

	// Export in requested format
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// HandleExportReport exports a report in specific format
func (h *Handler) HandleExportReport(w http.ResponseWriter, r *http.Request, scanID, format string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get scan
	scan, err := h.scanStore.GetScan(scanID)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	// Convert scan results to findings
	findings := h.scanResultsToFindings(scan.Results)

	// Generate report
	gen := reports.NewGenerator()
	report := gen.GenerateReport(
		"Assessment Client",
		scan.Target,
		[]string{scan.Target},
		"security-assessment",
		scanID,
		findings,
		0,
		scan.StartedAt,
	)

	// Export in requested format
	switch format {
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="report_%s.html"`, scanID))
		renderer := reports.NewHTMLRenderer("", "#667eea")
		htmlContent := renderer.Render(report)
		w.Write([]byte(htmlContent))

	case "markdown", "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="report_%s.md"`, scanID))
		mdExp := reports.NewMarkdownExporter()
		mdContent := mdExp.Export(report, "")
		w.Write([]byte(mdContent))

	case "json":
		fallthrough
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="report_%s.json"`, scanID))
		jsonExp := reports.NewJSONExporter()
		jsonContent, _ := jsonExp.ExportString(report)
		w.Write([]byte(jsonContent))
	}
}

// scanResultsToFindings converts scan results to report findings
func (h *Handler) scanResultsToFindings(results []models.ToolOutput) []*reports.Finding {
	findings := make([]*reports.Finding, 0, len(results))

	for i, result := range results {
		finding := &reports.Finding{
			ID:          fmt.Sprintf("F%d", i+1),
			Type:        "vulnerability",
			Category:    result.ToolName,
			Title:       fmt.Sprintf("%s finding from %s", result.ToolName, result.Target),
			Severity:    "medium", // TODO: parse from tool output
			CVSSV3:      "7.5",     // placeholder
			CWE:         "CWE-Unknown",
			OWASP:       "A01:2021 – Broken Access Control",
			Target:      result.Target,
			Parameter:   "",
			Description: result.Stdout,
			Impact:      fmt.Sprintf("Potential issue detected by %s", result.ToolName),
			Evidence: reports.FindingEvidence{
				Tool:            result.ToolName,
				ResponseSnippet: result.Stdout,
			},
			Remediation: reports.RemediationSteps{
				Priority: "medium",
				Steps: []string{
					fmt.Sprintf("Review output from %s", result.ToolName),
					"Investigate finding and verify impact",
					"Re-test after remediation",
				},
				EstimatedEffort: "2-3 hours",
				Verification:    "Run tool again and verify fix",
			},
		}
		findings = append(findings, finding)
	}

	return findings
}

// getPriority maps severity to remediation priority
func (h *Handler) getPriority(severity string) string {
	switch severity {
	case "critical":
		return "immediate"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "low"
	}
}
