package model

// CompletionResult là kết quả sau khi complete
type CompletionResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
	LatencyMs    int64
}
