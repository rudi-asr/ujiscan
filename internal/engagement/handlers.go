// Package engagement provides HTTP handlers
package engagement

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Handler provides HTTP handlers for engagement endpoints
type Handler struct {
	engService *EngagementService
	engStore   EngagementStore
	findStore  FindingStore
	commStore  CommentStore
}

// NewHandler creates a new engagement handler
func NewHandler(engService *EngagementService, engStore EngagementStore, findStore FindingStore, commStore CommentStore) *Handler {
	return &Handler{
		engService: engService,
		engStore:   engStore,
		findStore:  findStore,
		commStore:  commStore,
	}
}

// CreateEngagementRequest represents a create engagement request
type CreateEngagementRequest struct {
	ClientName  string    `json:"client_name"`
	ProjectName string    `json:"project_name"`
	Description string    `json:"description"`
	Scope       []string  `json:"scope"`
	AssignedTo  []string  `json:"assigned_to"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
}

// HandleCreateEngagement handles POST /api/engagements
func (h *Handler) HandleCreateEngagement(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateEngagementRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Create engagement
	eng, err := h.engService.CreateEngagement(
		req.ClientName,
		req.ProjectName,
		req.Description,
		req.AssignedTo,
		req.Scope,
		req.StartDate,
		req.EndDate,
		userID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create engagement: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(eng)
}

// HandleListEngagements handles GET /api/engagements
func (h *Handler) HandleListEngagements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all engagements
	engs, err := h.engStore.ListEngagements()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list engagements: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(engs)
}

// HandleGetEngagement handles GET /api/engagements/{id}
func (h *Handler) HandleGetEngagement(w http.ResponseWriter, r *http.Request, engagementID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng, err := h.engStore.GetEngagement(engagementID)
	if err != nil {
		http.Error(w, "engagement not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eng)
}

// HandleUpdateEngagement handles PUT /api/engagements/{id}
func (h *Handler) HandleUpdateEngagement(w http.ResponseWriter, r *http.Request, engagementID string) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eng, err := h.engStore.GetEngagement(engagementID)
	if err != nil {
		http.Error(w, "engagement not found", http.StatusNotFound)
		return
	}

	// Decode update fields
	var updates map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Apply updates
	if clientName, ok := updates["client_name"].(string); ok {
		eng.ClientName = clientName
	}
	if description, ok := updates["description"].(string); ok {
		eng.Description = description
	}

	eng.UpdatedAt = time.Now()

	err = h.engStore.UpdateEngagement(eng)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update engagement: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eng)
}

// HandleAssignPentester handles POST /api/engagements/{id}/assign
func (h *Handler) HandleAssignPentester(w http.ResponseWriter, r *http.Request, engagementID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	err = h.engService.AssignPentester(engagementID, req.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to assign pentester: %v", err), http.StatusBadRequest)
		return
	}

	eng, _ := h.engStore.GetEngagement(engagementID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eng)
}

// HandleActivateEngagement handles POST /api/engagements/{id}/activate
func (h *Handler) HandleActivateEngagement(w http.ResponseWriter, r *http.Request, engagementID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.engService.ActivateEngagement(engagementID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to activate engagement: %v", err), http.StatusBadRequest)
		return
	}

	eng, _ := h.engStore.GetEngagement(engagementID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eng)
}

// HandleCompleteEngagement handles POST /api/engagements/{id}/complete
func (h *Handler) HandleCompleteEngagement(w http.ResponseWriter, r *http.Request, engagementID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.engService.CompleteEngagement(engagementID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to complete engagement: %v", err), http.StatusBadRequest)
		return
	}

	eng, _ := h.engStore.GetEngagement(engagementID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eng)
}

// HandleSignOffEngagement handles POST /api/engagements/{id}/signoff
func (h *Handler) HandleSignOffEngagement(w http.ResponseWriter, r *http.Request, engagementID, userID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.engService.SignOffEngagement(engagementID, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to sign off engagement: %v", err), http.StatusBadRequest)
		return
	}

	eng, _ := h.engStore.GetEngagement(engagementID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eng)
}

// HandleListFindings handles GET /api/engagements/{id}/findings
func (h *Handler) HandleListFindings(w http.ResponseWriter, r *http.Request, engagementID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	findings, err := h.findStore.ListFindingsByEngagement(engagementID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list findings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(findings)
}

// HandleGetFinding handles GET /api/findings/{id}
func (h *Handler) HandleGetFinding(w http.ResponseWriter, r *http.Request, findingID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	finding, err := h.findStore.GetFinding(findingID)
	if err != nil {
		http.Error(w, "finding not found", http.StatusNotFound)
		return
	}

	// Get comments
	comments, _ := h.commStore.GetComments(findingID)
	finding.Comments = comments

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finding)
}

// HandleAddComment handles POST /api/findings/{id}/comments
func (h *Handler) HandleAddComment(w http.ResponseWriter, r *http.Request, findingID, userID, userEmail, userName string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	comment, err := h.engService.AddCommentToFinding(findingID, userID, userName, userEmail, req.Text)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to add comment: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// HandleMarkFindingAsVerified handles PUT /api/findings/{id}/verify
func (h *Handler) HandleMarkFindingAsVerified(w http.ResponseWriter, r *http.Request, findingID, userID, userName string) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.engService.MarkFindingAsVerified(findingID, userID, userName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to verify finding: %v", err), http.StatusBadRequest)
		return
	}

	finding, _ := h.findStore.GetFinding(findingID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finding)
}

// HandleMarkFindingAsFalsePositive handles PUT /api/findings/{id}/false-positive
func (h *Handler) HandleMarkFindingAsFalsePositive(w http.ResponseWriter, r *http.Request, findingID, userID, userName string) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	err = h.engService.MarkFindingAsFalsePositive(findingID, userID, userName, req.Reason)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to mark as false positive: %v", err), http.StatusBadRequest)
		return
	}

	finding, _ := h.findStore.GetFinding(findingID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finding)
}

// HandleApproveFinding handles PUT /api/findings/{id}/approve
func (h *Handler) HandleApproveFinding(w http.ResponseWriter, r *http.Request, findingID, userID, userName string) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.engService.ApproveFinding(findingID, userID, userName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to approve finding: %v", err), http.StatusBadRequest)
		return
	}

	finding, _ := h.findStore.GetFinding(findingID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finding)
}

// HandleGetFindingComments handles GET /api/findings/{id}/comments
func (h *Handler) HandleGetFindingComments(w http.ResponseWriter, r *http.Request, findingID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	comments, err := h.commStore.GetComments(findingID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get comments: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}
