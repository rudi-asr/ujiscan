// Package engagement provides engagement and finding management
package engagement

import (
	"log"
	"time"

	"github.com/rudi-asr/ujiscan/internal/persistence"
)

// StateSnapshot represents the complete serializable state of engagement data
type StateSnapshot struct {
	Engagements []*Engagement     `json:"engagements"`
	Findings    []*Finding        `json:"findings"`
	Comments    []*FindingComment `json:"comments"`
	SavedAt     time.Time         `json:"saved_at"`
}

// SaveState persists the full engagement state (engagements, findings, comments)
// to the persistence layer.
func SaveState(pm *persistence.PersistenceManager, engStore EngagementStore, findStore FindingStore, commStore CommentStore) error {
	if pm == nil {
		return nil // Persistence not available
	}

	engagements, err := engStore.ListEngagements()
	if err != nil {
		log.Printf("Warning: failed to list engagements for save: %v", err)
		engagements = []*Engagement{} // Empty list on error
	}

	findings, err := findStore.ListFindings()
	if err != nil {
		log.Printf("Warning: failed to list findings for save: %v", err)
		findings = []*Finding{} // Empty list on error
	}

	// Collect comments belonging to the persisted findings
	var comments []*FindingComment
	for _, f := range findings {
		if f == nil {
			continue
		}
		cs, err := commStore.GetComments(f.ID)
		if err != nil {
			log.Printf("Warning: failed to list comments for finding %s: %v", f.ID, err)
			continue
		}
		comments = append(comments, cs...)
	}

	snapshot := StateSnapshot{
		Engagements: engagements,
		Findings:    findings,
		Comments:    comments,
		SavedAt:     time.Now().UTC(),
	}

	if err := pm.SaveSnapshot("engagement_state", snapshot); err != nil {
		log.Printf("⚠️ Failed to save engagement state: %v", err)
		return err
	}

	log.Printf("✅ Saved engagement state (%d engagements, %d findings, %d comments)",
		len(engagements), len(findings), len(comments))
	return nil
}

// LoadState restores engagement state from persistence into the in-memory stores.
func LoadState(pm *persistence.PersistenceManager, engStore EngagementStore, findStore FindingStore, commStore CommentStore) error {
	if pm == nil {
		return nil // Persistence not available
	}

	var snapshot StateSnapshot
	err := pm.LoadSnapshot("engagement_state", &snapshot)
	if err != nil {
		log.Printf("⚠️ Failed to load engagement state: %v", err)
		return err
	}

	if len(snapshot.Engagements) == 0 && len(snapshot.Findings) == 0 && len(snapshot.Comments) == 0 {
		log.Printf("ℹ️  No saved engagement state found (first run)")
		return nil
	}

	// Restore stores (idempotent: replaces current in-memory contents)
	if err := engStore.RestoreEngagements(snapshot.Engagements); err != nil {
		log.Printf("⚠️ Failed to restore engagements: %v", err)
		return err
	}
	if err := findStore.RestoreFindings(snapshot.Findings); err != nil {
		log.Printf("⚠️ Failed to restore findings: %v", err)
		return err
	}
	if err := commStore.RestoreComments(snapshot.Comments); err != nil {
		log.Printf("⚠️ Failed to restore comments: %v", err)
		return err
	}

	log.Printf("✅ Loaded engagement state (%d engagements, %d findings, %d comments)",
		len(snapshot.Engagements), len(snapshot.Findings), len(snapshot.Comments))
	return nil
}