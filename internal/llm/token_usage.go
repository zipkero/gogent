package llm

import "time"

// TokenUsage 는 LLM 호출 시 소비된 토큰 정보와 호출 메타데이터를 담는다.
// Phase 9 비용 정책의 기반 데이터로 사용된다.
type TokenUsage struct {
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CalledAt         time.Time `json:"called_at"`
	RequestID        string    `json:"request_id,omitempty"`
}
