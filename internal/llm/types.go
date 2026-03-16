package llm

// Message 는 LLM 대화의 단일 메시지를 표현한다.
// Role: "system", "user", "assistant"
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest 는 LLM에 전달하는 요청 구조체다.
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// CompletionResponse 는 LLM으로부터 받은 응답 구조체다.
type CompletionResponse struct {
	Content      string     `json:"content"`
	FinishReason string     `json:"finish_reason"`
	Usage        TokenUsage `json:"usage"`
}

// TokenUsage 는 LLM 호출 시 소비된 토큰 정보를 담는다.
// Phase 3에서 비용 추적의 기반 데이터로 사용된다.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
