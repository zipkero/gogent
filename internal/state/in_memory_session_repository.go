package state

import (
	"context"
	"sync"
)

// InMemorySessionRepository 는 map 기반의 SessionRepository 구현체다.
// 프로세스 내 메모리에만 저장되므로 재시작 시 데이터가 소실된다.
// Redis 구현(RedisSessionRepository)으로 교체하기 전 동작 검증용으로 사용한다.
type InMemorySessionRepository struct {
	mu   sync.RWMutex
	data map[string]SessionState
}

// NewInMemorySessionRepository 는 초기화된 InMemorySessionRepository를 반환한다.
func NewInMemorySessionRepository() *InMemorySessionRepository {
	return &InMemorySessionRepository{
		data: make(map[string]SessionState),
	}
}

// Load 는 sessionID에 해당하는 SessionState를 반환한다.
// 존재하지 않으면 빈 SessionState와 nil error를 반환한다.
func (r *InMemorySessionRepository) Load(_ context.Context, sessionID string) (SessionState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, ok := r.data[sessionID]
	if !ok {
		return SessionState{}, nil
	}
	return state, nil
}

// Save 는 sessionID에 SessionState를 저장한다.
func (r *InMemorySessionRepository) Save(_ context.Context, sessionID string, state SessionState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[sessionID] = state
	return nil
}
