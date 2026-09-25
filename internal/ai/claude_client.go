package ai

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ClaudeClient implements AI Client interface using Claude
type ClaudeClient struct {
	config *AIConfig
	// In production, would use github.com/anthropics/sdk-go or similar
}

// NewClaudeClient creates a new Claude AI client
func NewClaudeClient(config *AIConfig) (*ClaudeClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	if config.Model == "" {
		config.Model = "claude-3-sonnet-20240229"
	}

	if config.Temperature == 0 {
		config.Temperature = 0.7
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 1024
	}

	return &ClaudeClient{
		config: config,
	}, nil
}

// Analyze analyzes findings using Claude
func (c *ClaudeClient) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
	// Build prompt
	_ = c.buildAnalysisPrompt(req)

	// In production, would call Claude API
	// For now, return simulated response
	response := &AnalysisResponse{
		Content:        fmt.Sprintf("Analysis of findings for %s: High severity vulnerabilities detected", req.Target),
		Severity:       8,
		Recommendation: "Immediate patching required",
		RiskLevel:      "HIGH",
		Model:          c.config.Model,
		Provider:       string(ProviderClaude),
		Timestamp:      time.Now(),
	}

	return response, nil
}

// EnrichFinding enriches a single finding
func (c *ClaudeClient) EnrichFinding(ctx context.Context, req *FindingEnrichmentRequest) (*FindingEnrichment, error) {
	// Build enrichment prompt
	_ = c.buildEnrichmentPrompt(req)

	// In production, would call Claude API
	// For now, return simulated enrichment
	enrichment := &FindingEnrichment{
		Severity:    8,
		CWE:         "CWE-79",
		Remediation: "Input validation and output encoding required",
		Category:    "Injection",
		Confidence:  0.95,
	}

	return enrichment, nil
}

// Chat sends a message and gets response
func (c *ClaudeClient) Chat(ctx context.Context, messages []Message) (string, error) {
	// In production, would call Claude API
	// For now, return simulated response
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}

	lastMessage := messages[len(messages)-1]
	return fmt.Sprintf("Response to: %s", lastMessage.Content), nil
}

// GetModel returns the model name
func (c *ClaudeClient) GetModel() string {
	return c.config.Model
}

// GetProvider returns the provider
func (c *ClaudeClient) GetProvider() AIProvider {
	return ProviderClaude
}

// Close closes resources
func (c *ClaudeClient) Close() error {
	return nil
}

// buildAnalysisPrompt builds the analysis prompt
func (c *ClaudeClient) buildAnalysisPrompt(req *AnalysisRequest) string {
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = `You are a security expert analyzing penetration test findings. 
Provide concise analysis with:
1. Severity (1-10)
2. Risk level (LOW/MEDIUM/HIGH/CRITICAL)
3. Remediation recommendations

Keep responses brief and actionable.`
	}

	userPrompt := fmt.Sprintf(`Analyze these findings for target %s:

%s

Provide structured analysis.`, req.Target, req.Findings)

	return fmt.Sprintf("SYSTEM: %s\n\nUSER: %s", systemPrompt, userPrompt)
}

// buildEnrichmentPrompt builds the enrichment prompt
func (c *ClaudeClient) buildEnrichmentPrompt(req *FindingEnrichmentRequest) string {
	return fmt.Sprintf(`Enrich this security finding:
Type: %s
Description: %s
Evidence: %s
Target: %s

Provide:
1. CVSS-like severity (1-10)
2. CWE ID if applicable
3. Remediation steps
4. Risk category

Keep response concise.`, req.FindingType, req.Description, req.Evidence, req.Target)
}

// ParseSeverity extracts severity from response
func (c *ClaudeClient) ParseSeverity(response string) int {
	// Simple parsing: look for "severity: X" or similar
	if strings.Contains(response, "severity") {
		parts := strings.Split(response, "severity")
		if len(parts) > 1 {
			rest := parts[1]
			// Look for first number
			for i := 0; i < len(rest); i++ {
				if rest[i] >= '0' && rest[i] <= '9' {
					num := string(rest[i])
					if i+1 < len(rest) && rest[i+1] >= '0' && rest[i+1] <= '9' {
						num = num + string(rest[i+1])
					}
					if val, err := strconv.Atoi(num); err == nil && val >= 1 && val <= 10 {
						return val
					}
				}
			}
		}
	}
	return 5 // Default middle value
}
