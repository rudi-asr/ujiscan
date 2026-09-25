// Package engagement provides engagement management
package engagement

import (
	"time"
)

// EngagementStatus represents the status of an engagement
type EngagementStatus string

const (
	StatusDraft      EngagementStatus = "draft"
	StatusActive     EngagementStatus = "active"
	StatusCompleted  EngagementStatus = "completed"
	StatusSignedOff  EngagementStatus = "signed_off"
	StatusArchived   EngagementStatus = "archived"
)

// Engagement represents a penetration testing engagement
type Engagement struct {
	ID              string              `json:"id"`
	ClientName      string              `json:"client_name"`
	ProjectName     string              `json:"project_name"`
	Description     string              `json:"description"`
	Scope           []string            `json:"scope"` // Targets (domains, IPs)
	Status          EngagementStatus    `json:"status"`
	
	// Team assignment
	AssignedTo      []string            `json:"assigned_to"` // User IDs
	CreatedBy       string              `json:"created_by"`  // User ID who created
	ApprovedBy      *string             `json:"approved_by,omitempty"` // Admin who approved findings
	DeliveredBy     *string             `json:"delivered_by,omitempty"` // User who delivered to client
	
	// Dates
	StartDate       time.Time           `json:"start_date"`
	EndDate         time.Time           `json:"end_date"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	CompletedAt     *time.Time          `json:"completed_at,omitempty"`
	SignedOffAt     *time.Time          `json:"signed_off_at,omitempty"`
	
	// Summary stats
	ScanCount       int                 `json:"scan_count"`
	FindingCount    int                 `json:"finding_count"`
	CriticalCount   int                 `json:"critical_count"`
	HighCount       int                 `json:"high_count"`
}

// FindingComment represents a comment on a finding
type FindingComment struct {
	ID        string    `json:"id"`
	FindingID string    `json:"finding_id"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	UserName  string    `json:"user_name"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// FindingStatus represents the status of a finding
type FindingStatus string

const (
	StatusOpen          FindingStatus = "open"
	StatusUnderReview   FindingStatus = "under_review"
	StatusFalsePositive FindingStatus = "false_positive"
	StatusVerified      FindingStatus = "verified"
	StatusDeferred      FindingStatus = "deferred"
	StatusFixed         FindingStatus = "fixed"
)

// Finding represents a security finding with team metadata
type Finding struct {
	ID                  string            `json:"id"`
	EngagementID        string            `json:"engagement_id"`
	ScanID              string            `json:"scan_id"`
	
	// Finding data
	Type                string            `json:"type"`
	Category            string            `json:"category"`
	Title               string            `json:"title"`
	Severity            string            `json:"severity"`
	CVSSV3              string            `json:"cvss_v3"`
	CWE                 string            `json:"cwe"`
	OWASP               string            `json:"owasp"`
	Target              string            `json:"target"`
	Parameter           string            `json:"parameter"`
	Description         string            `json:"description"`
	Impact              string            `json:"impact"`
	
	// Team workflow
	Status              FindingStatus     `json:"status"`
	Comments            []*FindingComment `json:"comments,omitempty"`
	FalsePositiveReason string            `json:"false_positive_reason,omitempty"`
	DiscoveredBy        string            `json:"discovered_by"` // User ID
	DiscoveredByName    string            `json:"discovered_by_name"`
	ReviewedBy          *string           `json:"reviewed_by,omitempty"` // User ID
	ReviewedByName      *string           `json:"reviewed_by_name,omitempty"`
	ApprovedBy          *string           `json:"approved_by,omitempty"` // User ID
	ApprovedByName      *string           `json:"approved_by_name,omitempty"`
	
	// Timeline
	DiscoveredAt        time.Time         `json:"discovered_at"`
	ReviewedAt          *time.Time        `json:"reviewed_at,omitempty"`
	ApprovedAt          *time.Time        `json:"approved_at,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// EngagementStore provides engagement persistence
type EngagementStore interface {
	CreateEngagement(eng *Engagement) error
	GetEngagement(id string) (*Engagement, error)
	UpdateEngagement(eng *Engagement) error
	DeleteEngagement(id string) error
	ListEngagements() ([]*Engagement, error)
	ListEngagementsByUser(userID string) ([]*Engagement, error)
	ListEngagementsByStatus(status EngagementStatus) ([]*Engagement, error)
	RestoreEngagements(engs []*Engagement) error
}

// FindingStore provides finding persistence
type FindingStore interface {
	CreateFinding(finding *Finding) error
	GetFinding(id string) (*Finding, error)
	UpdateFinding(finding *Finding) error
	DeleteFinding(id string) error
	ListFindings() ([]*Finding, error)
	ListFindingsByEngagement(engagementID string) ([]*Finding, error)
	ListFindingsByStatus(status FindingStatus) ([]*Finding, error)
	ListFindingsByScan(scanID string) ([]*Finding, error)
	RestoreFindings(findings []*Finding) error
}

// CommentStore provides comment persistence
type CommentStore interface {
	AddComment(comment *FindingComment) error
	GetComments(findingID string) ([]*FindingComment, error)
	DeleteComment(id string) error
	RestoreComments(comments []*FindingComment) error
}

// EngagementService provides business logic for engagements
type EngagementService struct {
	engStore  EngagementStore
	findStore FindingStore
	commStore CommentStore
}

// NewEngagementService creates a new engagement service
func NewEngagementService(engStore EngagementStore, findStore FindingStore, commStore CommentStore) *EngagementService {
	return &EngagementService{
		engStore:  engStore,
		findStore: findStore,
		commStore: commStore,
	}
}

// CreateEngagement creates a new engagement
func (es *EngagementService) CreateEngagement(clientName, projectName, description string, assignedTo []string, scope []string, startDate, endDate time.Time, createdBy string) (*Engagement, error) {
	now := time.Now()
	eng := &Engagement{
		ID:          "eng_" + now.Format("20060102150405"),
		ClientName:  clientName,
		ProjectName: projectName,
		Description: description,
		Scope:       scope,
		Status:      StatusDraft,
		AssignedTo:  assignedTo,
		CreatedBy:   createdBy,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return eng, es.engStore.CreateEngagement(eng)
}

// ActivateEngagement moves engagement from draft to active
func (es *EngagementService) ActivateEngagement(engagementID string) error {
	eng, err := es.engStore.GetEngagement(engagementID)
	if err != nil {
		return err
	}

	if eng.Status != StatusDraft {
		return ErrInvalidStatus
	}

	eng.Status = StatusActive
	eng.UpdatedAt = time.Now()
	return es.engStore.UpdateEngagement(eng)
}

// CompleteEngagement marks engagement as completed
func (es *EngagementService) CompleteEngagement(engagementID string) error {
	eng, err := es.engStore.GetEngagement(engagementID)
	if err != nil {
		return err
	}

	if eng.Status != StatusActive {
		return ErrInvalidStatus
	}

	eng.Status = StatusCompleted
	now := time.Now()
	eng.CompletedAt = &now
	eng.UpdatedAt = now
	return es.engStore.UpdateEngagement(eng)
}

// SignOffEngagement moves engagement to signed_off (final delivery)
func (es *EngagementService) SignOffEngagement(engagementID, userID string) error {
	eng, err := es.engStore.GetEngagement(engagementID)
	if err != nil {
		return err
	}

	if eng.Status != StatusCompleted {
		return ErrInvalidStatus
	}

	eng.Status = StatusSignedOff
	eng.DeliveredBy = &userID
	now := time.Now()
	eng.SignedOffAt = &now
	eng.UpdatedAt = now
	return es.engStore.UpdateEngagement(eng)
}

// AssignPentester assigns a pentester to an engagement
func (es *EngagementService) AssignPentester(engagementID, userID string) error {
	eng, err := es.engStore.GetEngagement(engagementID)
	if err != nil {
		return err
	}

	// Check if already assigned
	for _, id := range eng.AssignedTo {
		if id == userID {
			return ErrAlreadyAssigned
		}
	}

	eng.AssignedTo = append(eng.AssignedTo, userID)
	eng.UpdatedAt = time.Now()
	return es.engStore.UpdateEngagement(eng)
}

// UnassignPentester removes a pentester from an engagement
func (es *EngagementService) UnassignPentester(engagementID, userID string) error {
	eng, err := es.engStore.GetEngagement(engagementID)
	if err != nil {
		return err
	}

	// Remove user from list
	newAssigned := make([]string, 0)
	for _, id := range eng.AssignedTo {
		if id != userID {
			newAssigned = append(newAssigned, id)
		}
	}

	eng.AssignedTo = newAssigned
	eng.UpdatedAt = time.Now()
	return es.engStore.UpdateEngagement(eng)
}

// AddCommentToFinding adds a comment to a finding
func (es *EngagementService) AddCommentToFinding(findingID, userID, userName, email, text string) (*FindingComment, error) {
	// Verify finding exists
	_, err := es.findStore.GetFinding(findingID)
	if err != nil {
		return nil, err
	}

	comment := &FindingComment{
		ID:        "cmt_" + time.Now().Format("20060102150405"),
		FindingID: findingID,
		UserID:    userID,
		UserEmail: email,
		UserName:  userName,
		Text:      text,
		CreatedAt: time.Now(),
	}

	return comment, es.commStore.AddComment(comment)
}

// MarkFindingAsVerified marks a finding as verified
func (es *EngagementService) MarkFindingAsVerified(findingID, reviewerID, reviewerName string) error {
	finding, err := es.findStore.GetFinding(findingID)
	if err != nil {
		return err
	}

	if finding.Status == StatusFalsePositive {
		return ErrAlreadyMarkedFalsePositive
	}

	finding.Status = StatusVerified
	finding.ReviewedBy = &reviewerID
	finding.ReviewedByName = &reviewerName
	now := time.Now()
	finding.ReviewedAt = &now
	finding.UpdatedAt = now

	return es.findStore.UpdateFinding(finding)
}

// MarkFindingAsFalsePositive marks a finding as false positive with reason
func (es *EngagementService) MarkFindingAsFalsePositive(findingID, reviewerID, reviewerName, reason string) error {
	finding, err := es.findStore.GetFinding(findingID)
	if err != nil {
		return err
	}

	finding.Status = StatusFalsePositive
	finding.FalsePositiveReason = reason
	finding.ReviewedBy = &reviewerID
	finding.ReviewedByName = &reviewerName
	now := time.Now()
	finding.ReviewedAt = &now
	finding.UpdatedAt = now

	return es.findStore.UpdateFinding(finding)
}

// ApproveFinding marks finding as officially approved by senior
func (es *EngagementService) ApproveFinding(findingID, approverID, approverName string) error {
	finding, err := es.findStore.GetFinding(findingID)
	if err != nil {
		return err
	}

	if finding.Status == StatusFalsePositive {
		return ErrCannotApproveFalsePositive
	}

	if finding.Status != StatusVerified && finding.Status != StatusOpen {
		return ErrInvalidFindingStatus
	}

	finding.Status = StatusVerified
	finding.ApprovedBy = &approverID
	finding.ApprovedByName = &approverName
	now := time.Now()
	finding.ApprovedAt = &now
	finding.UpdatedAt = now

	return es.findStore.UpdateFinding(finding)
}

// DeferFinding defers a finding to a later date
func (es *EngagementService) DeferFinding(findingID string) error {
	finding, err := es.findStore.GetFinding(findingID)
	if err != nil {
		return err
	}

	finding.Status = StatusDeferred
	finding.UpdatedAt = time.Now()

	return es.findStore.UpdateFinding(finding)
}

// GetFindingComments retrieves all comments for a finding
func (es *EngagementService) GetFindingComments(findingID string) ([]*FindingComment, error) {
	return es.commStore.GetComments(findingID)
}
