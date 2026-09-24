// Package audit provides audit logging
package audit

import (
	"time"
)

// Action represents an auditable action
type Action string

const (
	// User actions
	ActionUserLogin           Action = "user_login"
	ActionUserLogout          Action = "user_logout"
	ActionUserCreate          Action = "user_create"
	ActionUserDelete          Action = "user_delete"
	ActionPasswordChange      Action = "password_change"

	// Scan actions
	ActionScanCreate          Action = "scan_create"
	ActionScanStart           Action = "scan_start"
	ActionScanComplete        Action = "scan_complete"
	ActionScanDelete          Action = "scan_delete"

	// Finding actions
	ActionFindingCreate       Action = "finding_create"
	ActionFindingVerify       Action = "finding_verify"
	ActionFindingMarkFP       Action = "finding_mark_false_positive"
	ActionFindingApprove      Action = "finding_approve"
	ActionFindingDelete       Action = "finding_delete"

	// Engagement actions
	ActionEngagementCreate    Action = "engagement_create"
	ActionEngagementActivate  Action = "engagement_activate"
	ActionEngagementComplete  Action = "engagement_complete"
	ActionEngagementSignOff   Action = "engagement_signoff"
	ActionEngagementAssign    Action = "engagement_assign_pentester"

	// Comment actions
	ActionCommentAdd          Action = "comment_add"
	ActionCommentDelete       Action = "comment_delete"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
	Email     string                 `json:"email"`
	UserName  string                 `json:"user_name"`
	Action    Action                 `json:"action"`
	Resource  string                 `json:"resource"` // /scan/scan_123, /finding/f123, etc
	Changes   map[string]interface{} `json:"changes,omitempty"` // What changed
	Details   string                 `json:"details,omitempty"` // Human-readable description
	IPAddress string                 `json:"ip_address,omitempty"`
	Status    string                 `json:"status"` // "success", "failure"
	ErrorMsg  string                 `json:"error_msg,omitempty"`
}

// AuditStore provides audit log persistence
type AuditStore interface {
	Log(log *AuditLog) error
	GetLog(id string) (*AuditLog, error)
	ListLogs() ([]*AuditLog, error)
	ListLogsByUser(userID string) ([]*AuditLog, error)
	ListLogsByAction(action Action) ([]*AuditLog, error)
	ListLogsByDateRange(startTime, endTime time.Time) ([]*AuditLog, error)
	ListLogsByResource(resource string) ([]*AuditLog, error)
	DeleteOldLogs(olderThan time.Time) error
}

// AuditService provides audit logging operations
type AuditService struct {
	store AuditStore
}

// NewAuditService creates a new audit service
func NewAuditService(store AuditStore) *AuditService {
	return &AuditService{
		store: store,
	}
}

// LogAction logs an action
func (as *AuditService) LogAction(userID, email, userName, ipAddress string, action Action, resource string, details string) error {
	log := &AuditLog{
		ID:        "log_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
		UserID:    userID,
		Email:     email,
		UserName:  userName,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		Status:    "success",
	}

	return as.store.Log(log)
}

// LogActionWithChanges logs an action with changes
func (as *AuditService) LogActionWithChanges(userID, email, userName, ipAddress string, action Action, resource string, details string, changes map[string]interface{}) error {
	log := &AuditLog{
		ID:        "log_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
		UserID:    userID,
		Email:     email,
		UserName:  userName,
		Action:    action,
		Resource:  resource,
		Changes:   changes,
		Details:   details,
		IPAddress: ipAddress,
		Status:    "success",
	}

	return as.store.Log(log)
}

// LogError logs a failed action
func (as *AuditService) LogError(userID, email, userName, ipAddress string, action Action, resource string, errorMsg string) error {
	log := &AuditLog{
		ID:        "log_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
		UserID:    userID,
		Email:     email,
		UserName:  userName,
		Action:    action,
		Resource:  resource,
		IPAddress: ipAddress,
		Status:    "failure",
		ErrorMsg:  errorMsg,
	}

	return as.store.Log(log)
}

// GetLogs retrieves logs with optional filters
func (as *AuditService) GetLogs() ([]*AuditLog, error) {
	return as.store.ListLogs()
}

// GetLogsByUser retrieves logs for a specific user
func (as *AuditService) GetLogsByUser(userID string) ([]*AuditLog, error) {
	return as.store.ListLogsByUser(userID)
}

// GetLogsByAction retrieves logs for a specific action
func (as *AuditService) GetLogsByAction(action Action) ([]*AuditLog, error) {
	return as.store.ListLogsByAction(action)
}

// GetLogsByResource retrieves logs for a specific resource
func (as *AuditService) GetLogsByResource(resource string) ([]*AuditLog, error) {
	return as.store.ListLogsByResource(resource)
}

// GetLogsByDateRange retrieves logs within a date range
func (as *AuditService) GetLogsByDateRange(startTime, endTime time.Time) ([]*AuditLog, error) {
	return as.store.ListLogsByDateRange(startTime, endTime)
}

// PruneOldLogs removes logs older than specified time
func (as *AuditService) PruneOldLogs(olderThan time.Time) error {
	return as.store.DeleteOldLogs(olderThan)
}
