package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rudi-asr/ujiscan/internal/models"
)

// Provider display names
var ProviderNames = map[string]string{
	string(ProviderDeepSeek): "DeepSeek Flash",
	string(ProviderOpenAI):   "OpenAI",
	string(ProviderClaude):   "Claude",
	string(ProviderOpenCode): "OpenCode Zen",
}

// Client handles AI-driven security analysis via 3 providers
type Client struct {
	provider string
	apiKey   string
	model    string
	endpoint string
	orgID    string

	// AgentKnowledgeLoader — otak .md rahasia (AGENTS.md + skills + rules)
	knowledge *AgentKnowledgeLoader

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
	Evidence    string `json:"evidence,omitempty"`
}

// NewClient creates new AI client with default provider (deepseek)
func NewClient() *Client {
	return NewClientWithProvider(string(ProviderDeepSeek))
}

// NewClientWithProvider creates AI client for a specific provider.
// Supported providers: deepseek, openai, claude.
// Falls back to deepseek for unknown providers.
func NewClientWithProvider(provider string) *Client {
	if provider == "" {
		provider = string(ProviderDeepSeek)
	}
	if provider != string(ProviderDeepSeek) && provider != string(ProviderOpenAI) && provider != string(ProviderClaude) {
		fmt.Printf("[ai] Unknown provider %q, falling back to %s\n", provider, ProviderDeepSeek)
		provider = string(ProviderDeepSeek)
	}

	var apiKey, model, endpoint string
	switch provider {
	case string(ProviderOpenAI):
		apiKey = os.Getenv("OPENAI_API_KEY")
		model = "gpt-4o-mini"
		endpoint = "https://api.openai.com/v1/chat/completions"
	case string(ProviderClaude):
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		model = "claude-sonnet-4-5-20250929"
		endpoint = "https://api.anthropic.com/v1/messages"
	default: // deepseek
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
		model = "deepseek-chat" // DeepSeek Flash (V3)
		endpoint = "https://api.deepseek.com/chat/completions"
	}

	if apiKey == "" {
		fmt.Printf("[ai] WARNING: %s API key not set (dari env %s), AI pakai mock mode\n",
			ProviderNames[provider], envVarFor(provider))
	}

	return &Client{
		provider:  provider,
		apiKey:    apiKey,
		model:     model,
		endpoint:  endpoint,
		orgID:     os.Getenv("OPENAI_ORG_ID"),
		feedback:  make(map[string]*ToolFeedback),
		knowledge: NewAgentKnowledgeLoader("agents"),
	}
}

// Provider returns the current provider name
func (c *Client) Provider() string { return c.provider }

// ProviderEnvVar returns the env var name for a provider
func envVarFor(provider string) string {
	switch provider {
	case string(ProviderOpenAI):
		return "OPENAI_API_KEY"
	case string(ProviderClaude):
		return "ANTHROPIC_API_KEY"
	default:
		return "DEEPSEEK_API_KEY"
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
	// System prompt dari .md rahasia (skill fase + rules + master)
	sysPrompt, _ := c.knowledge.BuildSystemPrompt("vulnscan", objective)

	prompt := sysPrompt + fmt.Sprintf(`

You are an expert penetration tester analyzing security scan results.

Tool: %s
Objective: %s
Output:
%s

Based on this output, provide:
1. Analysis of findings
2. List of recommended next tools to run (only tool names from the tools available for this phase)
3. Reasoning for recommendations
4. Confidence level (0.0-1.0)
5. Whether scan should continue or stop

Respond in JSON format only with keys: analysis, recommended_tools, reasoning, confidence, stop_scan.`, toolName, objective, truncateOutput(output, 2000))

	decision, err := c.callProvider(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	return decision, nil
}

// ExtractVulnerabilities extracts structured vulnerabilities from tool output
func (c *Client) ExtractVulnerabilities(ctx context.Context, toolName string, output string) ([]VulnerabilityAssessment, error) {
	// System prompt dari .md (skill vulnscan + rules) — konsisten severity
	sysPrompt, _ := c.knowledge.BuildSystemPrompt("vulnscan", "")

	// Seleksi skill vuln DINAMIS berdasarkan tool/konteks output (hemat token):
	// hanya skill yang relevan dengan tool pemindai + skill global (header/ssl).
	relevant := map[string]string{
		"nuclei":   "ujiscan-vuln-xss,ujiscan-vuln-sqli,ujiscan-vuln-ssrf,ujiscan-vuln-lfi,ujiscan-vuln-ssti,ujiscan-vuln-idor,ujiscan-vuln-open-redirect",
		"nikto":    "ujiscan-vuln-header,ujiscan-vuln-lfi,ujiscan-vuln-ssti",
		"sslscan":  "ujiscan-vuln-ssl",
		"testssl":  "ujiscan-vuln-ssl",
		"nmap":     "ujiscan-vuln-ssl,ujiscan-vuln-header",
		"whatweb":  "ujiscan-vuln-header",
		"httpx":    "ujiscan-vuln-header,ujiscan-vuln-open-redirect",
		"gobuster": "ujiscan-vuln-lfi,ujiscan-vuln-header",
		"ffuf":     "ujiscan-vuln-sqli,ujiscan-vuln-xss,ujiscan-vuln-lfi",
		"sqlmap":   "ujiscan-vuln-sqli",
		"dalfox":   "ujiscan-vuln-xss",
		"gitleaks": "", // no dedicated skill
	}
	skillNames := []string{}
	if list, ok := relevant[strings.ToLower(toolName)]; ok {
		for _, s := range strings.Split(list, ",") {
			if s != "" {
				skillNames = append(skillNames, s)
			}
		}
	}
	// Fallback: jika tool tak dikenal, masukkan skill inti (xss,sqli,lfi,ssrf)
	if len(skillNames) == 0 {
		skillNames = []string{"ujiscan-vuln-xss", "ujiscan-vuln-sqli", "ujiscan-vuln-lfi", "ujiscan-vuln-ssrf"}
	}

	skillCorpus := ""
	for _, skill := range skillNames {
		if content, err := c.knowledge.LoadSkillNama(skill); err == nil {
			skillCorpus += "\n" + content
		}
	}

	prompt := sysPrompt + skillCorpus + fmt.Sprintf(`
Extract security vulnerabilities from %s output.
Output:
%s

For each vulnerability, provide:
- title: concise vulnerability name
- severity: CRITICAL, HIGH, MEDIUM, LOW, INFO (sesuai panduan di atas)
- description: what was found
- impact: potential business impact
- tool_source: %s
- evidence: bukti singkat dari output (wajib ada)

Return as JSON array. If no vulnerabilities found, return empty array.`,
		toolName, truncateOutput(output, 2000), toolName)

	resp, err := c.callProviderRaw(ctx, prompt)
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
	// Fase awal = recon → system prompt dari .md
	sysPrompt, _ := c.knowledge.BuildSystemPrompt("recon", "")

	prompt := sysPrompt + fmt.Sprintf(`
As a penetration tester, recommend security scanning tools for:
Target: %s
Scan Type: %s

Available tools (dari katalog ujiscan): %s

Return JSON with:
{
  "tools": ["tool1", "tool2", ...],
  "reasoning": "why these tools"
}

Optimize for efficiency - recommend 3-5 most relevant tools first.`,
		target, scanType, strings.Join(c.knowledge.LoadToolCatalogForPhase("recon"), ", "))

	resp, err := c.callProviderRaw(ctx, prompt)
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

// callProvider makes the provider API call and returns a structured decision
func (c *Client) callProvider(ctx context.Context, prompt string) (*AgentDecision, error) {
	resp, err := c.callProviderRaw(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var decision AgentDecision
	if err := json.Unmarshal([]byte(resp), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &decision, nil
}

// callProviderRaw makes the actual provider API request.
// Falls back to intelligent mock when API key is not configured.
func (c *Client) callProviderRaw(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		// Fallback: return intelligent mock response
		return c.mockOpenAIResponse(prompt)
	}

	switch c.provider {
	case string(ProviderOpenAI):
		return c.callOpenAIReal(ctx, prompt)
	case string(ProviderClaude):
		return c.callClaudeReal(ctx, prompt)
	default:
		return c.callDeepSeekReal(ctx, prompt)
	}
}

// ---- OpenAI (https://api.openai.com/v1/chat/completions) ----

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	MaxTokens int            `json:"max_tokens,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) callOpenAIReal(ctx context.Context, prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "system", Content: "You are an expert penetration testing assistant. Always respond with valid JSON only."},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 1024,
	}
	return c.httpChatCall(ctx, reqBody, "Authorization")
}

// ---- DeepSeek (https://api.deepseek.com/chat/completions - OpenAI compatible) ----

func (c *Client) callDeepSeekReal(ctx context.Context, prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "system", Content: "You are an expert penetration testing assistant. Always respond with valid JSON only."},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 1024,
	}
	return c.httpChatCall(ctx, reqBody, "Authorization")
}

// httpChatCall shared OpenAI-compatible endpoint call (OpenAI + DeepSeek)
func (c *Client) httpChatCall(ctx context.Context, reqBody openAIRequest, authHeader string) (string, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(authHeader, "Bearer "+c.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("provider request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("provider API error %d: %s", resp.StatusCode, truncateOutput(string(respBody), 300))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse provider response: %w", err)
	}
	if apiResp.Error != nil {
		return "", fmt.Errorf("provider error: %s", apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// ---- Claude (https://api.anthropic.com/v1/messages) ----

type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system"`
	Messages    []openAIMessage `json:"messages"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) callClaudeReal(ctx context.Context, prompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    "You are an expert penetration testing assistant. Always respond with valid JSON only.",
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("claude request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude API error %d: %s", resp.StatusCode, truncateOutput(string(respBody), 300))
	}

	var apiResp claudeResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse claude response: %w", err)
	}
	if apiResp.Error != nil {
		return "", fmt.Errorf("claude error: %s", apiResp.Error.Message)
	}
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("claude returned no content")
	}

	return apiResp.Content[0].Text, nil
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