package report

import "math"

// Deduplicator removes duplicate findings
type Deduplicator struct {
	config DeduplicationConfig
}

// NewDeduplicator creates a new deduplicator
func NewDeduplicator(config DeduplicationConfig) *Deduplicator {
	return &Deduplicator{
		config: config,
	}
}

// Deduplicate removes duplicate findings from a list
func (d *Deduplicator) Deduplicate(findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	// Keep unique findings
	unique := []Finding{}
	seen := make(map[string]bool)

	for _, f := range findings {
		isDuplicate := false

		// Check against existing findings
		for _, existing := range unique {
			if d.isSimilar(f, existing) {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			unique = append(unique, f)
			seen[f.ID] = true
		}
	}

	return unique
}

// isSimilar checks if two findings are similar
func (d *Deduplicator) isSimilar(f1, f2 Finding) bool {
	similarity := d.calculateSimilarity(f1, f2)
	return similarity >= d.config.SimilarityThreshold
}

// calculateSimilarity computes similarity between two findings (0-1)
func (d *Deduplicator) calculateSimilarity(f1, f2 Finding) float64 {
	score := 0.0
	weights := 0.0

	// Type matching (30% weight)
	if d.config.MatchType {
		if f1.Type == f2.Type {
			score += 0.3
		}
		weights += 0.3
	}

	// Target matching (30% weight)
	if d.config.MatchTarget {
		if f1.Target == f2.Target {
			score += 0.3
		}
		weights += 0.3
	}

	// Evidence similarity (40% weight)
	if d.config.MatchEvidence {
		evidenceSim := d.stringSimilarity(f1.Evidence, f2.Evidence)
		score += 0.4 * evidenceSim
		weights += 0.4
	}

	if weights == 0 {
		return 0
	}

	return score / weights
}

// stringSimilarity computes similarity between two strings (0-1)
func (d *Deduplicator) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Levenshtein distance
	dist := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	return 1.0 - (float64(dist) / maxLen)
}

// levenshteinDistance computes edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	runes1 := []rune(s1)
	runes2 := []rune(s2)

	m := len(runes1)
	n := len(runes2)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}

	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 0
			if runes1[i-1] != runes2[j-1] {
				cost = 1
			}

			dp[i][j] = min(
				dp[i-1][j]+1,      // deletion
				dp[i][j-1]+1,      // insertion
				dp[i-1][j-1]+cost, // substitution
			)
		}
	}

	return dp[m][n]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
