// Package auth provides authentication and authorization
package auth

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// Role represents user role in the system
type Role string

const (
	RoleAdmin      Role = "admin"
	RolePentester  Role = "pentester"
	RoleClient     Role = "client"
	RoleAuditor    Role = "auditor"
)

// User represents a system user
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	PasswordHash string `json:"-"` // Never expose in JSON
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

// UserStore provides user persistence
type UserStore interface {
	CreateUser(email, name, passwordHash string, role Role) (*User, error)
	GetUser(id string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id string) error
	ListUsers() ([]*User, error)
	UpdateLastLogin(id string) error
}

// Permission represents an action a user can perform
type Permission string

const (
	// Scan permissions
	PermCreateScan    Permission = "create_scan"
	PermViewScan      Permission = "view_scan"
	PermDeleteScan    Permission = "delete_scan"
	PermManageScan    Permission = "manage_scan"

	// Finding permissions
	PermViewFinding   Permission = "view_finding"
	PermEditFinding   Permission = "edit_finding"
	PermApproveFinding Permission = "approve_finding"

	// Engagement permissions
	PermCreateEngagement    Permission = "create_engagement"
	PermViewEngagement      Permission = "view_engagement"
	PermManageEngagement    Permission = "manage_engagement"
	PermApproveEngagement   Permission = "approve_engagement"
	PermDeliverEngagement   Permission = "deliver_engagement"

	// User management permissions
	PermManageUsers   Permission = "manage_users"
	PermViewAuditLog  Permission = "view_audit_log"
	PermViewStats    Permission = "view_stats"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		// Scans
		PermCreateScan, PermViewScan, PermDeleteScan, PermManageScan,
		// Findings
		PermViewFinding, PermEditFinding, PermApproveFinding,
		// Engagements
		PermCreateEngagement, PermViewEngagement, PermManageEngagement,
		PermApproveEngagement, PermDeliverEngagement,
		// Admin
		PermManageUsers, PermViewAuditLog, PermViewStats,
	},

	RolePentester: {
		// Scans
		PermCreateScan, PermViewScan, PermDeleteScan,
		// Findings
		PermViewFinding, PermEditFinding,
		// Engagements
		PermCreateEngagement, PermViewEngagement, PermManageEngagement,
		// Stats
		PermViewStats,
	},

	RoleClient: {
		// Can only view own engagement
		PermViewEngagement,
		// Can only view own findings (sanitized)
		PermViewFinding,
	},

	RoleAuditor: {
		// Can view everything, cannot modify
		PermViewScan, PermViewFinding, PermViewEngagement,
		PermViewAuditLog, PermViewStats,
	},
}

// HasPermission checks if a role has a permission
func HasPermission(role Role, permission Permission) bool {
	perms, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// PasswordHash hashes a password using SHA256
func PasswordHash(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) bool {
	return PasswordHash(password) == hash
}

// Claims represents JWT token claims
type Claims struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

// IsExpired checks if token claims are expired
func (c *Claims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// TokenManager handles JWT token creation and validation
type TokenManager interface {
	GenerateToken(user *User, duration time.Duration) (string, error)
	ValidateToken(token string) (*Claims, error)
	RefreshToken(token string) (string, error)
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"-"` // Never expose
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// SessionStore provides session persistence
type SessionStore interface {
	CreateSession(userID, token, ipAddress, userAgent string, expiresAt time.Time) (*Session, error)
	GetSession(sessionID string) (*Session, error)
	ValidateSession(sessionID string) (*Session, error)
	InvalidateSession(sessionID string) error
	InvalidateAllUserSessions(userID string) error
	ListUserSessions(userID string) ([]*Session, error)
}
