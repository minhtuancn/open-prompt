package providers

import "testing"

// TestAnthropicImplementsProvider kiểm tra AnthropicProvider implement Provider interface
func TestAnthropicImplementsProvider(t *testing.T) {
	var _ Provider = (*AnthropicProvider)(nil)
}

// TestOpenAIImplementsProvider kiểm tra OpenAIProvider implement Provider interface
func TestOpenAIImplementsProvider(t *testing.T) {
	var _ Provider = (*OpenAIProvider)(nil)
}

// TestOllamaImplementsProvider kiểm tra OllamaProvider implement Provider interface
func TestOllamaImplementsProvider(t *testing.T) {
	var _ Provider = (*OllamaProvider)(nil)
}
