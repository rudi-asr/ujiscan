package ai

import (
	"context"
	"testing"
	"time"
)

func TestNewClaudeClient(t *testing.T) {
	config := &AIConfig{
		Provider: ProviderClaude,
		APIKey:   "test-key",
	}

	client, err := NewClaudeClient(config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if client == nil {
		t.Errorf("Expected client to be created")
	}

	if client.GetModel() != "claude-3-sonnet-20240229" {
		t.Errorf("Expected default model to be set")
	}
}

func TestClaudeClientAnalyze(t *testing.T) {
	config := &AIConfig{
		Provider: ProviderClaude,
		APIKey:   "test-key",
	}

	client, _ := NewClaudeClient(config)

	req := &AnalysisRequest{
		Target:   "example.com",
		Findings: "XSS vulnerability found",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.Analyze(ctx, req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if response == nil {
		t.Errorf("Expected response")
	}

	if response.Severity < 1 || response.Severity > 10 {
		t.Errorf("Expected severity between 1-10, got %d", response.Severity)
	}
}

func TestClaudeClientEnrichFinding(t *testing.T) {
	config := &AIConfig{
		Provider: ProviderClaude,
		APIKey:   "test-key",
	}

	client, _ := NewClaudeClient(config)

	req := &FindingEnrichmentRequest{
		FindingType: "XSS",
		Description: "Reflected XSS in search parameter",
		Target:      "example.com",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	enrichment, err := client.EnrichFinding(ctx, req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if enrichment == nil {
		t.Errorf("Expected enrichment")
	}

	if enrichment.Severity < 1 || enrichment.Severity > 10 {
		t.Errorf("Expected severity 1-10, got %d", enrichment.Severity)
	}
}

func TestClaudeClientChat(t *testing.T) {
	config := &AIConfig{
		Provider: ProviderClaude,
		APIKey:   "test-key",
	}

	client, _ := NewClaudeClient(config)

	messages := []Message{
		{
			Role:    "user",
			Content: "What is the severity of this finding?",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.Chat(ctx, messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if response == "" {
		t.Errorf("Expected response")
	}
}

func TestHermesKeyManager(t *testing.T) {
	keyMgr := NewHermesKeyManager()

	if keyMgr == nil {
		t.Errorf("Expected key manager to be created")
	}

	// GetKey without env vars should return error
	_, err := keyMgr.GetClaudeKey()
	// We expect this to fail if CLAUDE_API_KEY is not set
	// That's fine - just test the error handling
	if err != nil && err.Error() == "" {
		t.Errorf("Expected error message")
	}
}

func TestAIConfigTypes(t *testing.T) {
	config := &AIConfig{
		Provider:    ProviderClaude,
		Model:       "claude-3",
		APIKey:      "test",
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	if config.Provider != ProviderClaude {
		t.Errorf("Expected Claude provider")
	}

	if config.MaxTokens != 1024 {
		t.Errorf("Expected max tokens 1024")
	}
}

func TestAnalysisRequestResponse(t *testing.T) {
	req := &AnalysisRequest{
		Target:   "test.com",
		Findings: "Test findings",
	}

	resp := &AnalysisResponse{
		Content:   "Analysis result",
		Severity:  8,
		RiskLevel: "HIGH",
		Model:     "claude-3",
		Provider:  "claude",
	}

	if req.Target != "test.com" {
		t.Errorf("Expected target test.com")
	}

	if resp.Severity != 8 {
		t.Errorf("Expected severity 8")
	}
}
