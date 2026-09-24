// Package auth provides HTTP handlers for authentication
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler provides HTTP handlers for auth endpoints
type Handler struct {
	authService *AuthService
	userStore   UserStore
}

// NewHandler creates a new auth handler
func NewHandler(authService *AuthService, userStore UserStore) *Handler {
	return &Handler{
		authService: authService,
		userStore:   userStore,
	}
}

// HandleLogin handles POST /auth/login
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Get client IP
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	userAgent := r.Header.Get("User-Agent")

	// Perform login
	resp, err := h.authService.Login(req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		http.Error(w, fmt.Sprintf("login failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleLogout handles POST /auth/logout
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get token from header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "missing authorization header", http.StatusUnauthorized)
		return
	}

	// Parse token
	var token string
	if _, err := fmt.Sscanf(authHeader, "Bearer %s", &token); err != nil {
		http.Error(w, "invalid authorization header", http.StatusUnauthorized)
		return
	}

	// Logout
	err := h.authService.Logout(token)
	if err != nil {
		http.Error(w, fmt.Sprintf("logout failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"logged out"}`)
}

// HandleMe handles GET /auth/me
func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user info from context
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user from store
	user, err := h.userStore.GetUser(userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Return user info (without password)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}

// ChangePasswordRequest represents a change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// HandleChangePassword handles POST /auth/change-password
func (h *Handler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Change password
	err = h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to change password: %v", err), http.StatusBadRequest)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"password changed"}`)
}

// HandleListUsers handles GET /api/users (admin only)
func (h *Handler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is admin
	role := GetRole(r)
	if role != RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Get all users
	users, err := h.userStore.ListUsers()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list users: %v", err), http.StatusInternalServerError)
		return
	}

	// Return users (without passwords)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// CreateUserRequest represents a create user request
type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// HandleCreateUser handles POST /api/users (admin only)
func (h *Handler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is admin
	role := GetRole(r)
	if role != RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate role
	userRole := Role(req.Role)
	if userRole != RoleAdmin && userRole != RolePentester && userRole != RoleClient && userRole != RoleAuditor {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	// Hash password
	passwordHash := PasswordHash(req.Password)

	// Create user
	user, err := h.userStore.CreateUser(req.Email, req.Name, passwordHash, userRole)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create user: %v", err), http.StatusBadRequest)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}

// DeleteUserRequest represents a delete user request
type DeleteUserRequest struct {
	UserID string `json:"user_id"`
}

// HandleDeleteUser handles DELETE /api/users/{id} (admin only)
func (h *Handler) HandleDeleteUser(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is admin
	role := GetRole(r)
	if role != RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Prevent self-deletion
	currentUserID := GetUserID(r)
	if currentUserID == userID {
		http.Error(w, "cannot delete your own account", http.StatusBadRequest)
		return
	}

	// Delete user
	err := h.userStore.DeleteUser(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete user: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"user deleted"}`)
}
