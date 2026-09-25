package agent

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Handler provides HTTP handlers for the agent manager API
// (POST /api/agents/submit, GET /api/agents/status, /api/agents/tasks...).
type Handler struct {
	manager *Manager
}

// NewHandler creates an agent API handler.
func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

// HandleSubmit handles POST /api/agents/submit
func (h *Handler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	if task.CreatedAt == 0 {
		task.CreatedAt = time.Now().Unix()
	}

	if err := h.manager.SubmitTask(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(task)
}

// HandleStatus handles GET /api/agents/status
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.manager.GetStatus())
}

// HandleListTasks handles GET /api/agents/tasks
func (h *Handler) HandleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := h.manager.GetTasks()
	if tasks == nil {
		tasks = []*Task{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(tasks),
		"tasks": tasks,
	})
}

// HandleGetTask handles GET /api/agents/tasks/{id}
func (h *Handler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	task := h.manager.GetTask(r.PathValue("id"))
	if task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// HandleWaitTask handles GET /api/agents/tasks/{id}/wait (long poll)
// Query param: timeout (seconds, default 30, max 120).
func (h *Handler) HandleWaitTask(w http.ResponseWriter, r *http.Request) {
	timeout := 30
	if v := r.URL.Query().Get("timeout"); v != "" {
		if t, err := strconv.Atoi(v); err == nil && t > 0 {
			timeout = t
		}
	}
	if timeout > 120 {
		timeout = 120
	}

	taskID := r.PathValue("id")
	task, err := h.manager.WaitForTask(r.Context(), taskID, time.Duration(timeout)*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// HandleCancelTask handles DELETE /api/agents/tasks/{id}
func (h *Handler) HandleCancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if err := h.manager.CancelTask(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     taskID,
		"status": StatusTerminated,
	})
}
