// Package engagement provides in-memory storage
package engagement

import (
	"fmt"
	"sync"
)

// MemoryEngagementStore implements EngagementStore using in-memory storage
type MemoryEngagementStore struct {
	mu          sync.RWMutex
	engagements map[string]*Engagement
}

// NewMemoryEngagementStore creates a new in-memory engagement store
func NewMemoryEngagementStore() *MemoryEngagementStore {
	return &MemoryEngagementStore{
		engagements: make(map[string]*Engagement),
	}
}

// CreateEngagement creates a new engagement
func (s *MemoryEngagementStore) CreateEngagement(eng *Engagement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.engagements[eng.ID]; exists {
		return fmt.Errorf("engagement already exists")
	}

	s.engagements[eng.ID] = eng
	return nil
}

// GetEngagement gets an engagement by ID
func (s *MemoryEngagementStore) GetEngagement(id string) (*Engagement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	eng, exists := s.engagements[id]
	if !exists {
		return nil, ErrEngagementNotFound
	}

	return eng, nil
}

// UpdateEngagement updates an engagement
func (s *MemoryEngagementStore) UpdateEngagement(eng *Engagement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.engagements[eng.ID]; !exists {
		return ErrEngagementNotFound
	}

	s.engagements[eng.ID] = eng
	return nil
}

// DeleteEngagement deletes an engagement
func (s *MemoryEngagementStore) DeleteEngagement(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.engagements[id]; !exists {
		return ErrEngagementNotFound
	}

	delete(s.engagements, id)
	return nil
}

// ListEngagements lists all engagements
func (s *MemoryEngagementStore) ListEngagements() ([]*Engagement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	engs := make([]*Engagement, 0, len(s.engagements))
	for _, eng := range s.engagements {
		engs = append(engs, eng)
	}

	return engs, nil
}

// ListEngagementsByUser lists engagements assigned to a user
func (s *MemoryEngagementStore) ListEngagementsByUser(userID string) ([]*Engagement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	engs := make([]*Engagement, 0)
	for _, eng := range s.engagements {
		for _, assignedID := range eng.AssignedTo {
			if assignedID == userID {
				engs = append(engs, eng)
				break
			}
		}
	}

	return engs, nil
}

// ListEngagementsByStatus lists engagements by status
func (s *MemoryEngagementStore) ListEngagementsByStatus(status EngagementStatus) ([]*Engagement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	engs := make([]*Engagement, 0)
	for _, eng := range s.engagements {
		if eng.Status == status {
			engs = append(engs, eng)
		}
	}

	return engs, nil
}

// MemoryFindingStore implements FindingStore using in-memory storage
type MemoryFindingStore struct {
	mu       sync.RWMutex
	findings map[string]*Finding
}

// NewMemoryFindingStore creates a new in-memory finding store
func NewMemoryFindingStore() *MemoryFindingStore {
	return &MemoryFindingStore{
		findings: make(map[string]*Finding),
	}
}

// CreateFinding creates a new finding
func (s *MemoryFindingStore) CreateFinding(finding *Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.findings[finding.ID]; exists {
		return fmt.Errorf("finding already exists")
	}

	s.findings[finding.ID] = finding
	return nil
}

// GetFinding gets a finding by ID
func (s *MemoryFindingStore) GetFinding(id string) (*Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	finding, exists := s.findings[id]
	if !exists {
		return nil, ErrFindingNotFound
	}

	return finding, nil
}

// UpdateFinding updates a finding
func (s *MemoryFindingStore) UpdateFinding(finding *Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.findings[finding.ID]; !exists {
		return ErrFindingNotFound
	}

	s.findings[finding.ID] = finding
	return nil
}

// DeleteFinding deletes a finding
func (s *MemoryFindingStore) DeleteFinding(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.findings[id]; !exists {
		return ErrFindingNotFound
	}

	delete(s.findings, id)
	return nil
}

// ListFindings lists all findings
func (s *MemoryFindingStore) ListFindings() ([]*Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	findings := make([]*Finding, 0, len(s.findings))
	for _, f := range s.findings {
		findings = append(findings, f)
	}

	return findings, nil
}

// ListFindingsByEngagement lists findings for an engagement
func (s *MemoryFindingStore) ListFindingsByEngagement(engagementID string) ([]*Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	findings := make([]*Finding, 0)
	for _, f := range s.findings {
		if f.EngagementID == engagementID {
			findings = append(findings, f)
		}
	}

	return findings, nil
}

// ListFindingsByStatus lists findings by status
func (s *MemoryFindingStore) ListFindingsByStatus(status FindingStatus) ([]*Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	findings := make([]*Finding, 0)
	for _, f := range s.findings {
		if f.Status == status {
			findings = append(findings, f)
		}
	}

	return findings, nil
}

// ListFindingsByScan lists findings for a scan
func (s *MemoryFindingStore) ListFindingsByScan(scanID string) ([]*Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	findings := make([]*Finding, 0)
	for _, f := range s.findings {
		if f.ScanID == scanID {
			findings = append(findings, f)
		}
	}

	return findings, nil
}

// MemoryCommentStore implements CommentStore using in-memory storage
type MemoryCommentStore struct {
	mu       sync.RWMutex
	comments map[string][]*FindingComment // findingID -> comments
}

// NewMemoryCommentStore creates a new in-memory comment store
func NewMemoryCommentStore() *MemoryCommentStore {
	return &MemoryCommentStore{
		comments: make(map[string][]*FindingComment),
	}
}

// AddComment adds a comment to a finding
func (s *MemoryCommentStore) AddComment(comment *FindingComment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.comments[comment.FindingID]; !exists {
		s.comments[comment.FindingID] = make([]*FindingComment, 0)
	}

	s.comments[comment.FindingID] = append(s.comments[comment.FindingID], comment)
	return nil
}

// GetComments gets all comments for a finding
func (s *MemoryCommentStore) GetComments(findingID string) ([]*FindingComment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	comments, exists := s.comments[findingID]
	if !exists {
		return make([]*FindingComment, 0), nil
	}

	return comments, nil
}

// DeleteComment deletes a comment
func (s *MemoryCommentStore) DeleteComment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and delete comment
	for findingID, comments := range s.comments {
		for i, comment := range comments {
			if comment.ID == id {
				// Remove comment from slice
				s.comments[findingID] = append(comments[:i], comments[i+1:]...)
				return nil
			}
		}
	}

	return fmt.Errorf("comment not found")
}
