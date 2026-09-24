// Package dashboard provides dashboard service
package dashboard

import (
	"sync"
	"time"
)

// MemoryDashboardService implements DashboardService
type MemoryDashboardService struct {
	mu sync.RWMutex
}

// NewMemoryDashboardService creates a new dashboard service
func NewMemoryDashboardService() *MemoryDashboardService {
	return &MemoryDashboardService{}
}

// GetTeamDashboard returns the team dashboard
func (mds *MemoryDashboardService) GetTeamDashboard() (*TeamDashboard, error) {
	mds.mu.RLock()
	defer mds.mu.RUnlock()

	// Placeholder implementation - will be populated from real data
	return &TeamDashboard{
		TotalScans:          0,
		ActiveScans:         0,
		CompletedScans:      0,
		TotalFindings:       0,
		CriticalFindings:    0,
		HighFindings:        0,
		MediumFindings:      0,
		LowFindings:         0,
		InfoFindings:        0,
		VerifiedFindings:    0,
		FalsePositives:      0,
		PendingApprovals:    0,
		TotalEngagements:    0,
		ActiveEngagements:   0,
		CompletedEngagements: 0,
		TeamMembers:         make([]TeamMember, 0),
		RecentActivity:      make([]ActivityItem, 0),
		TopFindings:         make([]FindingItem, 0),
		UpcomingDeadlines:   make([]DeadlineItem, 0),
		UpdatedAt:           time.Now(),
	}, nil
}

// GetClientDashboard returns the client dashboard
func (mds *MemoryDashboardService) GetClientDashboard(engagementID string) (*ClientDashboard, error) {
	mds.mu.RLock()
	defer mds.mu.RUnlock()

	// Placeholder implementation - will be populated from real data
	return &ClientDashboard{
		EngagementID:        engagementID,
		ClientName:          "",
		ProjectName:         "",
		Status:              "",
		TotalFindings:       0,
		CriticalFindings:    0,
		HighFindings:        0,
		MediumFindings:      0,
		LowFindings:         0,
		InfoFindings:        0,
		RiskScore:           0,
		RiskLevel:           "LOW",
		Findings:            make([]ClientFinding, 0),
		ExecutiveSummary:    "",
		Recommendations:     make([]string, 0),
		UpdatedAt:           time.Now(),
	}, nil
}

// GetAdminDashboard returns the admin dashboard
func (mds *MemoryDashboardService) GetAdminDashboard() (*AdminDashboard, error) {
	mds.mu.RLock()
	defer mds.mu.RUnlock()

	// Placeholder implementation - will be populated from real data
	return &AdminDashboard{
		TotalUsers:          0,
		ActiveUsers:         0,
		TotalEngagements:    0,
		TotalScans:          0,
		TotalFindings:       0,
		RecentAuditLogs:     make([]AuditLogItem, 0),
		SystemHealth: SystemHealth{
			Uptime:           "0h0m0s",
			CPUUsage:         0.0,
			MemoryUsage:      0.0,
			DatabaseStatus:   "healthy",
			CacheStatus:      "healthy",
			BackupStatus:     "pending",
		},
		TopTeamMembers:   make([]TeamMember, 0),
		EngagementMetrics: make([]EngagementMetric, 0),
		UpdatedAt:        time.Now(),
	}, nil
}

// MemoryNotificationService implements NotificationService
type MemoryNotificationService struct {
	mu            sync.RWMutex
	notifications map[string][]*Notification
	index         map[string]*Notification
}

// NewMemoryNotificationService creates a new notification service
func NewMemoryNotificationService() *MemoryNotificationService {
	return &MemoryNotificationService{
		notifications: make(map[string][]*Notification),
		index:         make(map[string]*Notification),
	}
}

// SendNotification sends a notification
func (mns *MemoryNotificationService) SendNotification(userID string, notification *Notification) error {
	mns.mu.Lock()
	defer mns.mu.Unlock()

	if notification.ID == "" {
		notification.ID = "notif_" + time.Now().Format("20060102150405")
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}

	mns.notifications[userID] = append(mns.notifications[userID], notification)
	mns.index[notification.ID] = notification

	return nil
}

// GetNotifications retrieves notifications for a user
func (mns *MemoryNotificationService) GetNotifications(userID string) ([]*Notification, error) {
	mns.mu.RLock()
	defer mns.mu.RUnlock()

	notifs, exists := mns.notifications[userID]
	if !exists {
		return make([]*Notification, 0), nil
	}

	result := make([]*Notification, len(notifs))
	copy(result, notifs)

	return result, nil
}

// MarkAsRead marks a notification as read
func (mns *MemoryNotificationService) MarkAsRead(notificationID string) error {
	mns.mu.Lock()
	defer mns.mu.Unlock()

	notif, exists := mns.index[notificationID]
	if !exists {
		return ErrNotificationNotFound
	}

	notif.Read = true
	return nil
}

// DeleteNotification deletes a notification
func (mns *MemoryNotificationService) DeleteNotification(notificationID string) error {
	mns.mu.Lock()
	defer mns.mu.Unlock()

	_, exists := mns.index[notificationID]
	if !exists {
		return ErrNotificationNotFound
	}

	// Remove from index
	delete(mns.index, notificationID)

	// Remove from user's notifications
	for userID, notifs := range mns.notifications {
		for i, n := range notifs {
			if n.ID == notificationID {
				mns.notifications[userID] = append(notifs[:i], notifs[i+1:]...)
				break
			}
		}
	}

	return nil
}
