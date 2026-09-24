// Package dashboard provides dashboard models and services
package dashboard

import (
	"time"
)

// DashboardType represents the type of dashboard
type DashboardType string

const (
	DashboardTeam   DashboardType = "team"
	DashboardClient DashboardType = "client"
	DashboardAdmin  DashboardType = "admin"
)

// TeamDashboard represents the team dashboard
type TeamDashboard struct {
	TotalScans          int               `json:"total_scans"`
	ActiveScans         int               `json:"active_scans"`
	CompletedScans      int               `json:"completed_scans"`
	TotalFindings       int               `json:"total_findings"`
	CriticalFindings    int               `json:"critical_findings"`
	HighFindings        int               `json:"high_findings"`
	MediumFindings      int               `json:"medium_findings"`
	LowFindings         int               `json:"low_findings"`
	InfoFindings        int               `json:"info_findings"`
	VerifiedFindings    int               `json:"verified_findings"`
	FalsePositives      int               `json:"false_positives"`
	PendingApprovals    int               `json:"pending_approvals"`
	TotalEngagements    int               `json:"total_engagements"`
	ActiveEngagements   int               `json:"active_engagements"`
	CompletedEngagements int              `json:"completed_engagements"`
	TeamMembers         []TeamMember      `json:"team_members"`
	RecentActivity      []ActivityItem    `json:"recent_activity"`
	TopFindings         []FindingItem     `json:"top_findings"`
	UpcomingDeadlines   []DeadlineItem    `json:"upcoming_deadlines"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// ClientDashboard represents the client-facing dashboard
type ClientDashboard struct {
	EngagementID        string            `json:"engagement_id"`
	ClientName          string            `json:"client_name"`
	ProjectName         string            `json:"project_name"`
	Status              string            `json:"status"`
	TotalFindings       int               `json:"total_findings"`
	CriticalFindings    int               `json:"critical_findings"`
	HighFindings        int               `json:"high_findings"`
	MediumFindings      int               `json:"medium_findings"`
	LowFindings         int               `json:"low_findings"`
	InfoFindings        int               `json:"info_findings"`
	RiskScore           int               `json:"risk_score"`
	RiskLevel           string            `json:"risk_level"`
	Findings            []ClientFinding   `json:"findings"`
	ExecutiveSummary    string            `json:"executive_summary"`
	Recommendations     []string          `json:"recommendations"`
	StartDate           time.Time         `json:"start_date"`
	EndDate             time.Time         `json:"end_date"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// AdminDashboard represents the admin dashboard
type AdminDashboard struct {
	TotalUsers          int                    `json:"total_users"`
	ActiveUsers         int                    `json:"active_users"`
	TotalEngagements    int                    `json:"total_engagements"`
	TotalScans          int                    `json:"total_scans"`
	TotalFindings       int                    `json:"total_findings"`
	RecentAuditLogs     []AuditLogItem         `json:"recent_audit_logs"`
	SystemHealth        SystemHealth           `json:"system_health"`
	TopTeamMembers      []TeamMember           `json:"top_team_members"`
	EngagementMetrics   []EngagementMetric     `json:"engagement_metrics"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// TeamMember represents a team member on the dashboard
type TeamMember struct {
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Scans       int       `json:"scans"`
	Findings    int       `json:"findings"`
	LastActive  time.Time `json:"last_active"`
}

// ActivityItem represents a recent activity on the dashboard
type ActivityItem struct {
	Timestamp   time.Time `json:"timestamp"`
	UserName    string    `json:"user_name"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	Details     string    `json:"details"`
}

// FindingItem represents a finding on the dashboard
type FindingItem struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Severity      string    `json:"severity"`
	Status        string    `json:"status"`
	DiscoveredAt  time.Time `json:"discovered_at"`
	DiscoveredBy  string    `json:"discovered_by"`
}

// DeadlineItem represents an upcoming deadline
type DeadlineItem struct {
	EngagementID string    `json:"engagement_id"`
	ProjectName  string    `json:"project_name"`
	DueDate      time.Time `json:"due_date"`
	DaysRemaining int      `json:"days_remaining"`
}

// ClientFinding represents a finding for the client
type ClientFinding struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	Severity     string      `json:"severity"`
	Category     string      `json:"category"`
	Description  string      `json:"description"`
	Impact       string      `json:"impact"`
	Remediation  string      `json:"remediation"`
	References   []string    `json:"references"`
}

// AuditLogItem represents an audit log on the dashboard
type AuditLogItem struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	UserName  string    `json:"user_name"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	Status    string    `json:"status"`
}

// SystemHealth represents system health metrics
type SystemHealth struct {
	Uptime           string  `json:"uptime"`
	CPUUsage         float64 `json:"cpu_usage"`
	MemoryUsage      float64 `json:"memory_usage"`
	DatabaseStatus   string  `json:"database_status"`
	CacheStatus      string  `json:"cache_status"`
	LastBackup       time.Time `json:"last_backup"`
	BackupStatus     string  `json:"backup_status"`
}

// EngagementMetric represents engagement metrics
type EngagementMetric struct {
	EngagementID string    `json:"engagement_id"`
	ClientName   string    `json:"client_name"`
	ProjectName  string    `json:"project_name"`
	Status       string    `json:"status"`
	Findings     int       `json:"findings"`
	Duration     string    `json:"duration"`
	TeamSize     int       `json:"team_size"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// DashboardService provides dashboard operations
type DashboardService interface {
	GetTeamDashboard() (*TeamDashboard, error)
	GetClientDashboard(engagementID string) (*ClientDashboard, error)
	GetAdminDashboard() (*AdminDashboard, error)
}

// Notification represents a notification
type Notification struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	Type      string        `json:"type"` // "finding", "engagement", "approval", "comment"
	Title     string        `json:"title"`
	Message   string        `json:"message"`
	Resource  string        `json:"resource"`
	Read      bool          `json:"read"`
	CreatedAt time.Time     `json:"created_at"`
	Channel   string        `json:"channel"` // "email", "slack", "in-app"
}

// NotificationService provides notification operations
type NotificationService interface {
	SendNotification(userID string, notification *Notification) error
	GetNotifications(userID string) ([]*Notification, error)
	MarkAsRead(notificationID string) error
	DeleteNotification(notificationID string) error
}
