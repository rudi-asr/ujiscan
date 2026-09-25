package ai

import (
	"fmt"
	"os"
)

// HermesKeyManager retrieves API keys from Hermes secrets
type HermesKeyManager struct{}

// NewHermesKeyManager creates a new Hermes key manager
func NewHermesKeyManager() *HermesKeyManager {
	return &HermesKeyManager{}
}

// GetClaudeKey retrieves Claude API key from Hermes
func (m *HermesKeyManager) GetClaudeKey() (string, error) {
	// Try environment variable first
	key := os.Getenv("CLAUDE_API_KEY")
	if key != "" {
		return key, nil
	}

	// Try Hermes secrets (in production, would use Hermes SDK)
	// For now, check environment variables that Hermes might set
	key = os.Getenv("HERMES_CLAUDE_KEY")
	if key != "" {
		return key, nil
	}

	return "", fmt.Errorf("Claude API key not found in Hermes secrets or environment")
}

// GetOpenAIKey retrieves OpenAI API key from Hermes
func (m *HermesKeyManager) GetOpenAIKey() (string, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key != "" {
		return key, nil
	}

	key = os.Getenv("HERMES_OPENAI_KEY")
	if key != "" {
		return key, nil
	}

	return "", fmt.Errorf("OpenAI API key not found in Hermes secrets")
}

// GetDeepSeekKey retrieves DeepSeek API key from Hermes
func (m *HermesKeyManager) GetDeepSeekKey() (string, error) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key != "" {
		return key, nil
	}

	key = os.Getenv("HERMES_DEEPSEEK_KEY")
	if key != "" {
		return key, nil
	}

	return "", fmt.Errorf("DeepSeek API key not found in Hermes secrets")
}

// GetKey retrieves API key for specific provider
func (m *HermesKeyManager) GetKey(provider AIProvider) (string, error) {
	switch provider {
	case ProviderClaude:
		return m.GetClaudeKey()
	case ProviderOpenAI:
		return m.GetOpenAIKey()
	case ProviderDeepSeek:
		return m.GetDeepSeekKey()
	default:
		return "", fmt.Errorf("unknown provider: %v", provider)
	}
}

// GetAIConfig returns complete AI config with credentials from Hermes
func (m *HermesKeyManager) GetAIConfig(provider AIProvider, model string) (*AIConfig, error) {
	key, err := m.GetKey(provider)
	if err != nil {
		return nil, err
	}

	config := &AIConfig{
		Provider:    provider,
		APIKey:      key,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	// Set provider-specific defaults
	switch provider {
	case ProviderClaude:
		if model == "" {
			config.Model = "claude-3-sonnet-20240229"
		}
	case ProviderOpenAI:
		if model == "" {
			config.Model = "gpt-4"
		}
	case ProviderDeepSeek:
		if model == "" {
			config.Model = "deepseek-chat"
		}
		config.Endpoint = "https://api.deepseek.com/v1"
	}

	return config, nil
}
