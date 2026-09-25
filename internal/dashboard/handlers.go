// Package dashboard provides HTTP handlers
package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/rudi-asr/ujiscan/internal/auth"
)

// Handler provides HTTP handlers for dashboard endpoints
type Handler struct {
	dashboardService   DashboardService
	notificationService NotificationService
}

// NewHandler creates a new dashboard handler
func NewHandler(dashboardService DashboardService, notificationService NotificationService) *Handler {
	return &Handler{
		dashboardService:    dashboardService,
		notificationService: notificationService,
	}
}

// HandleTeamDashboard returns the team dashboard
// GET /api/dashboard/team
func (h *Handler) HandleTeamDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dashboard, err := h.dashboardService.GetTeamDashboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// HandleClientDashboard returns the client dashboard
// GET /api/dashboard/client/{engagementID}
func (h *Handler) HandleClientDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	engagementID := r.PathValue("engagementID")
	dashboard, err := h.dashboardService.GetClientDashboard(engagementID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// HandleAdminDashboard returns the admin dashboard
// GET /api/dashboard/admin
func (h *Handler) HandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dashboard, err := h.dashboardService.GetAdminDashboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// HandleGetNotifications returns notifications for a user
// GET /api/notifications
func (h *Handler) HandleGetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by auth middleware)
	userID := auth.GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notifications, err := h.notificationService.GetNotifications(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications": notifications,
	})
}

// HandleMarkNotificationAsRead marks a notification as read
// PUT /api/notifications/{id}/read
func (h *Handler) HandleMarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	notificationID := r.PathValue("id")
	err := h.notificationService.MarkAsRead(notificationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   notificationID,
		"read": true,
	})
}

// HandleDeleteNotification deletes a notification
// DELETE /api/notifications/{id}
func (h *Handler) HandleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	notificationID := r.PathValue("id")
	err := h.notificationService.DeleteNotification(notificationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
