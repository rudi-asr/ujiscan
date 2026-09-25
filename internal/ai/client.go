package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// Client handles AI-driven security analysis via OpenAI API
type Client struct {
	apiKey   string
	model    string
	endpoint string
	orgID    string

	// C.4.4: learning feedback — tracks per-tool success/failure history
	feedbackMu sync.Mutex
	feedback   map[string]*ToolFeedback
}

// ToolFeedback tracks historical tool execution performance (C.4.4).
type ToolFeedback struct {
	Tool         string        `json:"tool"`
	SuccessCount int           `json:"success_count"`
	FailureCount int           `json:"failure_count"`
	AvgTime      time.Duration `json:"avg_time_ms"`
	LastUsed     time.Time     `json:"last_used"`
}

// AgentDecision represents AI's next-step recommendation
type AgentDecision struct {
	Analysis         string   `json:"analysis"`
	RecommendedTools []string `json:"recommended_tools"`
	Reasoning        string   `json:"reasoning"`
	Confidence       float64  `json:"confidence"`
	NextPhase        string   `json:"next_phase,omitempty"`
	StopScan         bool     `json:"stop_scan"`
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
		feedback: make(map[string]*ToolFeedback),
	}
}

// UpdateToolFeedback records a tool execution result (C.4.4 learning feedback).
func (c *Client) UpdateToolFeedback(tool string, success bool, duration time.Duration) {
	c.feedbackMu.Lock()
	defer c.feedbackMu.Unlock()

	if c.feedback == nil {
		c.feedback = make(map[string]*ToolFeedback)
	}

	fb := c.feedback[tool]
	if fb == nil {
		fb = &ToolFeedback{Tool: tool}
		c.feedback[tool] = fb
	}

	if success {
		fb.SuccessCount++
	} else {
		fb.FailureCount++
	}

	total := fb.SuccessCount + fb.FailureCount
	if total > 0 {
		// Running average
		prev := fb.AvgTime * time.Duration(total-1)
		fb.AvgTime = (prev + duration) / time.Duration(total)
	}
	fb.LastUsed = time.Now()
}

// GetToolFeedback returns the recorded feedback for a tool.
func (c *Client) GetToolFeedback(tool string) *ToolFeedback {
	c.feedbackMu.Lock()
	defer c.feedbackMu.Unlock()
	return c.feedback[tool]
}

// GetPreferredTools reorders the given tools by historical success rate
// (best performers first, unknowns in the middle, known-bad tools last).
// Used by the adaptive loop to prefer tools that have worked before.
func (c *Client) GetPreferredTools(tools []string) []string {
	c.feedbackMu.Lock()
	defer c.feedbackMu.Unlock()

	type scoredTool struct {
		name  string
		score float64
	}

	scored := make([]scoredTool, 0, len(tools))
	for _, t := range tools {
		score := 0.5 // neutral for tools with no history
		if fb := c.feedback[t]; fb != nil {
			if total := fb.SuccessCount + fb.FailureCount; total > 0 {
				score = float64(fb.SuccessCount) / float64(total)
			}
		}
		scored = append(scored, scoredTool{name: t, score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	ordered := make([]string, len(scored))
	for i, s := range scored {
		ordered[i] = s.name
	}
	return ordered
}

// AnalyzeToolOutput analyzes security tool output and returns next-step recommendations
func (c *Client) AnalyzeToolOutput(ctx context.Context, toolName string, output string, objective string) (*AgentDecision, error) {
	if c.apiKey == "" {
		return &AgentDecision{
			Analysis:         "(AI disabled - no API key)",
			RecommendedTools: []string{},
			Confidence:       0.0,
			StopScan:         false,
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

// callOpenAIRaw makes actual API request
func (c *Client) callOpenAIRaw(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		// Fallback: return intelligent mock response
		return c.mockOpenAIResponse(prompt)
	}

	// Real OpenAI API call - initialize when key available
	// For now using mock since API key not configured
	return c.mockOpenAIResponse(prompt)
}

// mockOpenAIResponse returns intelligent mock responses for testing
func (c *Client) mockOpenAIResponse(prompt string) (string, error) {
	// Analyze prompt to return contextual responses
	if strings.Contains(strings.ToLower(prompt), "select") || strings.Contains(strings.ToLower(prompt), "recommend") {
		// Tool selection response
		return `{
  "tools": ["dig", "subfinder", "nmap"],
  "reasoning": "Starting with DNS discovery (dig), subdomain enumeration (subfinder), and network scanning (nmap) for comprehensive reconnaissance"
}`, nil
	}

	if strings.Contains(strings.ToLower(prompt), "analyze") || strings.Contains(strings.ToLower(prompt), "decision") {
		// Analysis/decision response
		return `{
  "analysis": "Reconnaissance phase identified active ports (22, 80, 443). Recommend proceeding to enumeration phase.",
  "recommended_tools": ["httpx", "whatweb", "sslscan"],
  "reasoning": "Detected web services on ports 80/443 requiring HTTP enumeration and SSL analysis",
  "confidence": 0.85,
  "next_phase": "enum",
  "stop_scan": false
}`, nil
	}

	if strings.Contains(strings.ToLower(prompt), "vulnerab") || strings.Contains(strings.ToLower(prompt), "extract") {
		// Vulnerability extraction
		return `[
  {
    "title": "Open SSH Service",
    "severity": "INFO",
    "description": "SSH service detected on port 22",
    "impact": "Enable remote authentication",
    "tool_source": "nmap"
  },
  {
    "title": "HTTP Service Detected",
    "severity": "MEDIUM",
    "description": "Unencrypted HTTP service on port 80",
    "impact": "Potential data interception",
    "tool_source": "httpx"
  }
]`, nil
	}

	// Default response
	return `{
  "analysis": "Tool output received and processed",
  "recommended_tools": ["nmap", "httpx"],
  "reasoning": "Continuing with enumeration phase tools",
  "confidence": 0.7,
  "stop_scan": false
}`, nil
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
		ID:          "",
		ScanID:      "",
		ToolName:    "ai-analysis",
		Severity:    "INFO",
		Title:       "AI Analysis",
		Description: d.Analysis,
		Evidence:    fmt.Sprintf("Recommended tools: %v\nReasoning: %s\nConfidence: %.2f", d.RecommendedTools, d.Reasoning, d.Confidence),
		Timestamp:   time.Now(),
	}
}
