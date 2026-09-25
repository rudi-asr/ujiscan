package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// Client handles AI-driven security analysis via OpenAI API
type Client struct {
	apiKey    string
	model     string
	endpoint  string
	orgID     string
}

// AgentDecision represents AI's next-step recommendation
type AgentDecision struct {
	Analysis       string   `json:"analysis"`
	RecommendedTools []string `json:"recommended_tools"`
	Reasoning      string   `json:"reasoning"`
	Confidence     float64  `json:"confidence"`
	NextPhase      string   `json:"next_phase,omitempty"`
	StopScan       bool     `json:"stop_scan"`
}

// VulnerabilityAssessment represents findings extracted by AI
type VulnerabilityAssessment struct {
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	ToolSource  string `json:"tool_source"`
}

// NewClient creates new AI client
func NewClient() *Client {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Fallback for testing - disabled in production
		fmt.Printf("[ai] WARNING: OPENAI_API_KEY not set, AI features disabled\n")
	}

	return &Client{
		apiKey:   apiKey,
		model:    "gpt-4o-mini", // Cost-effective model for security analysis
		endpoint: "https://api.openai.com/v1/chat/completions",
		orgID:    os.Getenv("OPENAI_ORG_ID"),
	}
}

// AnalyzeToolOutput analyzes security tool output and returns next-step recommendations
func (c *Client) AnalyzeToolOutput(ctx context.Context, toolName string, output string, objective string) (*AgentDecision, error) {
	if c.apiKey == "" {
		return &AgentDecision{
			Analysis:        "(AI disabled - no API key)",
			RecommendedTools: []string{},
			Confidence:      0.0,
			StopScan:        false,
		}, nil
	}

	// Create prompt for AI analysis
	prompt := fmt.Sprintf(`You are an expert penetration tester analyzing security scan results.

Tool: %s
Objective: %s
Output:
%s

Based on this output, provide:
1. Analysis of findings
2. List of recommended next tools to run (only tool names from: dig, subfinder, nmap, httpx, whatweb, sslscan, nuclei, nikto, gobuster)
3. Reasoning for recommendations
4. Confidence level (0.0-1.0)
5. Whether scan should continue or stop

Respond in JSON format only.`, toolName, objective, truncateOutput(output, 2000))

	decision, err := c.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	return decision, nil
}

// ExtractVulnerabilities extracts structured vulnerabilities from tool output
func (c *Client) ExtractVulnerabilities(ctx context.Context, toolName string, output string) ([]VulnerabilityAssessment, error) {
	if c.apiKey == "" {
		return []VulnerabilityAssessment{}, nil
	}

	prompt := fmt.Sprintf(`Extract security vulnerabilities from %s output.
Output:
%s

For each vulnerability, provide:
- title: concise vulnerability name
- severity: CRITICAL, HIGH, MEDIUM, LOW, INFO
- description: what was found
- impact: potential business impact
- tool_source: %s

Return as JSON array. If no vulnerabilities found, return empty array.`, 
		toolName, truncateOutput(output, 2000), toolName)

	resp, err := c.callOpenAIRaw(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("vulnerability extraction failed: %w", err)
	}

	var vulns []VulnerabilityAssessment
	if err := json.Unmarshal([]byte(resp), &vulns); err != nil {
		fmt.Printf("[ai] Failed to parse vulnerabilities JSON: %v\n", err)
		return []VulnerabilityAssessment{}, nil
	}

	return vulns, nil
}

// SelectToolsForTarget recommends initial tools based on target characteristics
func (c *Client) SelectToolsForTarget(ctx context.Context, target string, scanType string) ([]string, error) {
	if c.apiKey == "" {
		// Return default tool set
		return defaultToolsForType(scanType), nil
	}

	prompt := fmt.Sprintf(`As a penetration tester, recommend security scanning tools for:
Target: %s
Scan Type: %s

Available tools: dig, subfinder, nmap, httpx, whatweb, sslscan, nuclei, nikto, gobuster

Return JSON with:
{
  "tools": ["tool1", "tool2", ...],
  "reasoning": "why these tools"
}

Optimize for efficiency - recommend 3-5 most relevant tools first.`, target, scanType)

	resp, err := c.callOpenAIRaw(ctx, prompt)
	if err != nil {
		return defaultToolsForType(scanType), nil
	}

	var result struct {
		Tools     []string `json:"tools"`
		Reasoning string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return defaultToolsForType(scanType), nil
	}

	return result.Tools, nil
}

// ---- Internal Methods ----

// callOpenAI makes API call and returns structured decision
func (c *Client) callOpenAI(ctx context.Context, prompt string) (*AgentDecision, error) {
	resp, err := c.callOpenAIRaw(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var decision AgentDecision
	if err := json.Unmarshal([]byte(resp), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &decision, nil
}

// callOpenAIRaw makes actual API request (placeholder - real impl uses go-openai)
func (c *Client) callOpenAIRaw(ctx context.Context, prompt string) (string, error) {
	// TODO: Implement actual OpenAI API call using go-openai package
	// For now, return placeholder response for testing
	fmt.Printf("[ai] Prompt: %s\n", truncateOutput(prompt, 200))
	return `{"analysis":"Scan results analyzed","recommended_tools":["nmap","httpx"],"reasoning":"Standard recon workflow","confidence":0.8,"stop_scan":false}`, nil
}

// ---- Utility Functions ----

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

func defaultToolsForType(scanType string) []string {
	switch scanType {
	case "quick":
		return []string{"dig", "nmap"}
	case "regular":
		return []string{"dig", "subfinder", "nmap", "httpx", "whatweb"}
	case "full":
		return []string{"dig", "subfinder", "nmap", "httpx", "whatweb", "sslscan", "nuclei", "nikto"}
	default:
		return []string{"dig", "nmap"}
	}
}

// DefaultToolsForType returns default tools for scan type (exported version)
func DefaultToolsForType(scanType string) []string {
	return defaultToolsForType(scanType)
}

// ToModelsDecision converts AgentDecision to models.Finding for storage
func (d *AgentDecision) ToFinding() models.Finding {
	return models.Finding{
		ID:        "",
		ScanID:    "",
		ToolName:  "ai-analysis",
		Severity:  "INFO",
		Title:     "AI Analysis",
		Description: d.Analysis,
		Evidence: fmt.Sprintf("Recommended tools: %v\nReasoning: %s\nConfidence: %.2f", d.RecommendedTools, d.Reasoning, d.Confidence),
		Timestamp: time.Now(),
	}
}
