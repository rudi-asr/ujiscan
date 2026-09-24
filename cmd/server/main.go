package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudi-asr/ujiscan/internal/api"
	"github.com/rudi-asr/ujiscan/internal/executor"
	"github.com/rudi-asr/ujiscan/internal/playbook"
	"github.com/rudi-asr/ujiscan/internal/registry"
	"github.com/rudi-asr/ujiscan/internal/store"
	"github.com/rudi-asr/ujiscan/internal/tools"
)

const (
	Port = ":8081"
)

func Run() error {
	// Get project root from current working directory
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load registry from tools.yaml
	toolsYAML := filepath.Join(projectRoot, "tools.yaml")
	reg, err := registry.LoadRegistry(toolsYAML)
	if err != nil {
		log.Printf("Warning: Failed to load tools registry: %v", err)
		log.Printf("Using legacy tool executor instead")
	}

	// Initialize stores and executors
	scanStore := store.NewScanStore()
	toolExecutor := tools.NewExecutor()
	toolExecutor.InitializeDefaultTools()
	scanExecutor := executor.NewScanExecutor(scanStore, toolExecutor)

	// Create registry executor if registry loaded
	var regExecutor *tools.RegistryExecutor
	if reg != nil {
		regExecutor = tools.NewRegistryExecutor(reg)
		if err := regExecutor.ValidateRegistry(); err != nil {
			log.Printf("Warning: Registry validation failed: %v", err)
		} else {
			log.Printf("✅ Tool registry loaded successfully (%d tools, %d modes)", len(reg.Tools), len(reg.Modes))
		}
	}

	// Initialize playbook engine
	playbookLoader := playbook.NewPlaybookLoader(filepath.Join(projectRoot, "playbooks"))
	playbookEngine := playbook.NewEngine(playbookLoader, scanExecutor, scanStore)

	// Create API handler
	apiHandler := api.NewHandler(scanStore, toolExecutor, scanExecutor, playbookEngine)

	// Routes
	mux := http.NewServeMux()

	// Static files
	webDir := filepath.Join(projectRoot, "web")
	log.Printf("Web directory: %s", webDir)
	
	// Check if web directory exists
	if _, err := os.Stat(webDir); err != nil {
		return fmt.Errorf("web directory not found: %s", webDir)
	}
	
	mux.Handle("/", http.FileServer(http.Dir(webDir)))

	// API routes
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/tools", apiHandler.HandleListTools)
	mux.HandleFunc("/api/stats", apiHandler.HandleStats)
	mux.HandleFunc("/api/playbooks", apiHandler.HandleListPlaybooks)
	
	// Scan API routes with custom handler
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiHandler.HandleStartScan(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Playbook scan endpoint (supports both regular and agentic via query param or path)
	mux.HandleFunc("/api/scan/playbook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Check if this is agentic mode (by query param or path ending with /agentic)
			isAgentic := r.URL.Query().Get("agentic") == "true" || strings.HasSuffix(r.URL.Path, "/agentic")
			if isAgentic {
				apiHandler.HandleAgenticPlaybookScan(w, r)
			} else {
				apiHandler.HandlePlaybookScan(w, r)
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Alternative agentic endpoint for convenience
	mux.HandleFunc("/api/agentic", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiHandler.HandleAgenticPlaybookScan(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Dynamic scan routes: /api/scan/{id}
	// Must come AFTER specific routes like /api/scan/playbook
	mux.HandleFunc("/api/scan/", func(w http.ResponseWriter, r *http.Request) {
		// Extract scan ID from path: /api/scan/{id}
		// Skip if this is a playbook-specific route
		if strings.Contains(r.URL.Path, "/playbook") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/scan/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Scan ID required", http.StatusBadRequest)
			return
		}

		scanID := parts[0]

		switch r.Method {
		case http.MethodGet:
			apiHandler.HandleGetScan(w, r, scanID)
		case http.MethodDelete:
			apiHandler.HandleDeleteScan(w, r, scanID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Start server
	log.Printf("ujiscan server starting on http://localhost%s", Port)
	log.Printf("API endpoints:")
	log.Printf("  GET  /api/status         - Server health")
	log.Printf("  GET  /api/tools          - List tools")
	log.Printf("  GET  /api/stats          - Scan statistics")
	log.Printf("  GET  /api/playbooks      - List playbooks")
	log.Printf("  POST /api/scan           - Start scan")
	log.Printf("  POST /api/scan/playbook  - Start playbook scan")
	log.Printf("  GET  /api/scan/{id}      - Get scan details")
	log.Printf("  DEL  /api/scan/{id}      - Delete scan")
	log.Printf("CORS enabled for: http://localhost:8081, https://rudi-asr.github.io")

	// Wrap mux with CORS middleware
	corsHandler := corsMiddleware(mux)
	return http.ListenAndServe(Port, corsHandler)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"ujiscan"}`)
}

// corsMiddleware adds CORS headers to allow requests from GitHub Pages & localhost
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from GitHub Pages and localhost
		origin := r.Header.Get("Origin")
		if origin == "https://rudi-asr.github.io" || 
		   origin == "http://localhost:8081" || 
		   origin == "http://localhost:3000" ||
		   strings.HasPrefix(origin, "http://127.0.0.1") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")
		
		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
