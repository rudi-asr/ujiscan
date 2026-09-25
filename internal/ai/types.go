package ai

import "time"

// AIProvider represents AI service provider
type AIProvider string

const (
	ProviderClaude   AIProvider = "claude"
	ProviderOpenAI   AIProvider = "openai"
	ProviderDeepSeek AIProvider = "deepseek"
	ProviderOpenCode AIProvider = "opencode-zen"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "user", "assistant"
	Content string `json:"content"`
}

// AnalysisRequest represents a request to analyze findings
type AnalysisRequest struct {
	// Target being tested
	Target string `json:"target"`
	// Findings to analyze (as JSON string or text)
	Findings string `json:"findings"`
	// Context (e.g., engagement mode, target type)
	Context map[string]string `json:"context,omitempty"`
	// System prompt override
	SystemPrompt string `json:"system_prompt,omitempty"`
}

// AnalysisResponse represents AI analysis result
type AnalysisResponse struct {
	// Raw LLM response
	Content string `json:"content"`
	// Extracted severity score (1-10)
	Severity int `json:"severity,omitempty"`
	// Extracted recommendation
	Recommendation string `json:"recommendation,omitempty"`
	// Extracted risk level (LOW, MEDIUM, HIGH, CRITICAL)
	RiskLevel string `json:"risk_level,omitempty"`
	// Metadata
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	Timestamp time.Time `json:"timestamp"`
}

// FindingEnrichmentRequest is sent to AI for enrichment
type FindingEnrichmentRequest struct {
	FindingType string                 `json:"finding_type"`
	Description string                 `json:"description"`
	Evidence    string                 `json:"evidence,omitempty"`
	Target      string                 `json:"target"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// FindingEnrichment is AI-enriched finding data
type FindingEnrichment struct {
	// CVSS-like score (1-10)
	Severity int `json:"severity"`
	// CWE ID if applicable
	CWE string `json:"cwe,omitempty"`
	// Remediation steps
	Remediation string `json:"remediation,omitempty"`
	// Risk category
	Category string `json:"category,omitempty"`
	// Confidence (0.0-1.0)
	Confidence float64 `json:"confidence"`
}

// AIConfig represents AI client configuration
type AIConfig struct {
	Provider    AIProvider `json:"provider"`
	Model       string     `json:"model"`
	APIKey      string     `json:"api_key"`
	Endpoint    string     `json:"endpoint,omitempty"`
	Temperature float64    `json:"temperature"`
	MaxTokens   int        `json:"max_tokens"`
}
