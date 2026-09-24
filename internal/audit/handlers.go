// Package audit provides HTTP handlers
package audit

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler provides HTTP handlers for audit endpoints
type Handler struct {
	service *AuditService
}

// NewHandler creates a new audit handler
func NewHandler(service *AuditService) *Handler {
	return &Handler{
		service: service,
	}
}

// HandleListLogs returns all audit logs
// GET /api/audit/logs
func (h *Handler) HandleListLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs, err := h.service.GetLogs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
	})
}

// HandleListLogsByUser returns logs for a specific user
// GET /api/audit/user/{userID}
func (h *Handler) HandleListLogsByUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.PathValue("userID")
	logs, err := h.service.GetLogsByUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"logs":    logs,
	})
}

// HandleListLogsByAction returns logs for a specific action
// GET /api/audit/action/{action}
func (h *Handler) HandleListLogsByAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	actionStr := r.PathValue("action")
	action := Action(actionStr)

	logs, err := h.service.GetLogsByAction(action)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"action": action,
		"logs":   logs,
	})
}

// HandleListLogsByResource returns logs for a specific resource
// GET /api/audit/resource/{resource}
func (h *Handler) HandleListLogsByResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resource := r.PathValue("resource")
	logs, err := h.service.GetLogsByResource(resource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"resource": resource,
		"logs":     logs,
	})
}

// HandleListLogsByDateRange returns logs within a date range
// GET /api/audit/logs?start=2026-09-01&end=2026-09-30
func (h *Handler) HandleListLogsByDateRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "Invalid start time format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "Invalid end time format", http.StatusBadRequest)
		return
	}

	logs, err := h.service.GetLogsByDateRange(startTime, endTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"start": startTime,
		"end":   endTime,
		"logs":  logs,
	})
}

// HandleExportLogs exports audit logs as JSON
// GET /api/audit/export?format=json|csv
func (h *Handler) HandleExportLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	logs, err := h.service.GetLogs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.csv")

		// Write CSV header
		w.Write([]byte("ID,Timestamp,UserID,Email,UserName,Action,Resource,Details,Status,ErrorMsg\n"))

		// Write CSV rows
		for _, log := range logs {
			row := log.ID + "," +
				log.Timestamp.String() + "," +
				log.UserID + "," +
				log.Email + "," +
				log.UserName + "," +
				string(log.Action) + "," +
				log.Resource + "," +
				log.Details + "," +
				log.Status + "," +
				log.ErrorMsg + "\n"
			w.Write([]byte(row))
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"logs": logs,
		})
	}
}

// HandleStats returns audit statistics
// GET /api/audit/stats
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs, err := h.service.GetLogs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Count by status
	successCount := 0
	failureCount := 0

	// Count by action
	actionCount := make(map[Action]int)

	// Count by user
	userCount := make(map[string]int)

	for _, log := range logs {
		if log.Status == "success" {
			successCount++
		} else if log.Status == "failure" {
			failureCount++
		}

		actionCount[log.Action]++
		userCount[log.UserID]++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_logs":     len(logs),
		"success_count":  successCount,
		"failure_count":  failureCount,
		"action_count":   actionCount,
		"user_count":     userCount,
	})
}
