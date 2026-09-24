package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	// API health check
	mux.HandleFunc("/api/status", handleStatus)

	// Start server
	log.Printf("ujiscan server starting on http://localhost%s", Port)
	log.Printf("Serving static files from: %s", webDir)

	return http.ListenAndServe(Port, mux)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"ujiscan"}`)
}
