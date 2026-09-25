package ai

import "context"

// Client is the interface for AI services
type Client interface {
	// Analyze analyzes findings and returns structured response
	Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error)

	// EnrichFinding enriches a single finding with AI analysis
	EnrichFinding(ctx context.Context, req *FindingEnrichmentRequest) (*FindingEnrichment, error)

	// Chat sends a message and gets a response (generic chat)
	Chat(ctx context.Context, messages []Message) (string, error)

	// GetModel returns the model being used
	GetModel() string

	// GetProvider returns the provider
	GetProvider() AIProvider

	// Close closes any resources
	Close() error
}

// NewClient creates a new AI client based on configuration
func NewClient(config *AIConfig) (Client, error) {
	switch config.Provider {
	case ProviderClaude:
		return NewClaudeClient(config)
	case ProviderOpenAI:
		// TODO: Implement OpenAI client
		return NewClaudeClient(config) // Fallback to Claude for now
	case ProviderDeepSeek:
		// TODO: Implement DeepSeek client
		return NewClaudeClient(config) // Fallback to Claude for now
	default:
		return NewClaudeClient(config) // Default to Claude
	}
}
