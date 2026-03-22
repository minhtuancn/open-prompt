package model

// CompletionRequest là request gửi đến AI provider
type CompletionRequest struct {
	Model       string
	Prompt      string
	System      string
	Temperature float64
	MaxTokens   int
}

// CompletionResult là kết quả sau khi complete
type CompletionResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
	LatencyMs    int64
}
