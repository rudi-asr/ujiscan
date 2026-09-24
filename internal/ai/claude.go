package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// ClaudeClient wraps Claude API calls
type ClaudeClient struct {
	apiKey  string
	baseURL string
	model   string
}

// Message represents a message in Claude conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest is the request body for Claude API
type ClaudeRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Messages    []Message `json:"messages"`
	Temperature float64  `json:"temperature"`
}

// ClaudeResponse is the response from Claude API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(apiKey string) *ClaudeClient {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	return &ClaudeClient{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		model:   "claude-3-5-sonnet-20241022",
	}
}

// Chat sends a message to Claude and returns the response
func (c *ClaudeClient) Chat(messages []Message) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	req := ClaudeRequest{
		Model:       c.model,
		MaxTokens:   2048,
		Messages:    messages,
		Temperature: 0.7,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp ClaudeResponse
	err = json.Unmarshal(respBody, &apiResp)
	if err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return apiResp.Content[0].Text, nil
}

// AnalyzeToolOutput asks Claude to analyze tool output and decide next action
func (c *ClaudeClient) AnalyzeToolOutput(toolName string, output string, objective string) (string, error) {
	messages := []Message{
		{
			Role: "user",
			Content: fmt.Sprintf(`You are a penetration testing AI assistant. Analyze the following tool output and decide the next action.

Tool: %s
Tool Output:
%s

Objective: %s

Based on the output, provide a JSON response with:
- "analysis": brief analysis of what was found
- "recommendation": next action (continue, skip, exploit, or stop)
- "reasoning": why you recommend this action

Be concise. Return ONLY valid JSON.`, toolName, output, objective),
		},
	}

	return c.Chat(messages)
}

// PlanNextStep asks Claude to plan the next penetration testing step
func (c *ClaudeClient) PlanNextStep(scanContext string, objective string) (string, error) {
	messages := []Message{
		{
			Role: "user",
			Content: fmt.Sprintf(`You are a penetration testing AI. Based on the current scan context, plan the next step.

Objective: %s

Current Context:
%s

Provide a JSON response with:
- "next_phase": recommended next phase (recon, enum, exploit, verify, report)
- "reasoning": why this phase
- "expected_outcome": what we expect to find

Be strategic. Return ONLY valid JSON.`, objective, scanContext),
		},
	}

	return c.Chat(messages)
}
