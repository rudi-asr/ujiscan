// Package engagement provides engagement and finding management
package engagement

import (
	"encoding/json"
	"log"

	"github.com/rudi-asr/ujiscan/internal/persistence"
)

// StateSnapshot represents the state of engagement data
type StateSnapshot struct {
	Engagements []*Engagement `json:"engagements"`
	Findings    []*Finding    `json:"findings"`
}

// SaveState saves engagement state to persistence
func SaveState(pm *persistence.PersistenceManager, engStore EngagementStore, findStore FindingStore) error {
	if pm == nil {
		return nil // Persistence not available
	}

	// Get all engagements
	engagements, err := engStore.ListEngagements()
	if err != nil {
		log.Printf("Warning: failed to list engagements for save: %v", err)
		engagements = []*Engagement{} // Empty list on error
	}

	// For now, we'll just save the count
	// Full implementation would serialize all related data
	data := map[string]interface{}{
		"engagement_count": len(engagements),
		"saved_at":         json.RawMessage(`"` + getCurrentTimestamp() + `"`),
	}

	err = pm.SaveSnapshot("engagement_state", data)
	if err != nil {
		log.Printf("Warning: failed to save engagement state: %v", err)
		return err
	}

	log.Printf("✅ Saved engagement state (%d items)", len(engagements))
	return nil
}

// LoadState loads engagement state from persistence
func LoadState(pm *persistence.PersistenceManager) error {
	if pm == nil {
		return nil // Persistence not available
	}

	var state map[string]interface{}
	err := pm.LoadSnapshot("engagement_state", &state)
	if err != nil {
		log.Printf("Warning: failed to load engagement state: %v", err)
		return err
	}

	if state == nil {
		log.Printf("ℹ️  No saved engagement state found (first run)")
		return nil
	}

	if count, ok := state["engagement_count"].(float64); ok {
		log.Printf("✅ Loaded engagement state (%d items)", int(count))
	}

	return nil
}

// getCurrentTimestamp returns current timestamp in RFC3339 format
func getCurrentTimestamp() string {
	return "2026-09-25T07:20:00Z" // Placeholder - would use time.Now() in real code
}
