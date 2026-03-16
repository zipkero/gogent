package llm

import "context"

// LLMClient 는 LLM provider에 대한 추상화 인터페이스다.
// planner, reflector 등 LLM 호출이 필요한 컴포넌트는 이 인터페이스만 의존한다.
// OpenAI, Anthropic 등 구체 구현체는 이 인터페이스를 통해 주입된다.
type LLMClient interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
