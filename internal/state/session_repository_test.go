package state

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// sessionRepositorySuite 는 SessionRepository 구현체에 공통으로 적용되는 테스트 케이스다.
// InMemorySessionRepository와 RedisSessionRepository 모두 동일한 케이스를 통과해야 한다.
func sessionRepositorySuite(t *testing.T, repo SessionRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("Save_and_Load", func(t *testing.T) {
		want := SessionState{
			SessionID:     "sess-001",
			RecentContext: []string{"hello", "world"},
			ActiveGoal:    "test goal",
			LastUpdated:   time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
		}

		if err := repo.Save(ctx, want.SessionID, want); err != nil {
			t.Fatalf("Save: %v", err)
		}

		got, err := repo.Load(ctx, want.SessionID)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if got.SessionID != want.SessionID {
			t.Errorf("SessionID: got %q, want %q", got.SessionID, want.SessionID)
		}
		if got.ActiveGoal != want.ActiveGoal {
			t.Errorf("ActiveGoal: got %q, want %q", got.ActiveGoal, want.ActiveGoal)
		}
		if len(got.RecentContext) != len(want.RecentContext) {
			t.Errorf("RecentContext len: got %d, want %d", len(got.RecentContext), len(want.RecentContext))
		} else {
			for i := range want.RecentContext {
				if got.RecentContext[i] != want.RecentContext[i] {
					t.Errorf("RecentContext[%d]: got %q, want %q", i, got.RecentContext[i], want.RecentContext[i])
				}
			}
		}
		if !got.LastUpdated.Equal(want.LastUpdated) {
			t.Errorf("LastUpdated: got %v, want %v", got.LastUpdated, want.LastUpdated)
		}
	})

	t.Run("Load_nonexistent_returns_empty", func(t *testing.T) {
		got, err := repo.Load(ctx, "nonexistent-session-id")
		if err != nil {
			t.Fatalf("Load nonexistent: unexpected error: %v", err)
		}
		if got.SessionID != "" || got.ActiveGoal != "" || len(got.RecentContext) != 0 || !got.LastUpdated.IsZero() {
			t.Errorf("expected empty SessionState, got %+v", got)
		}
	})

	t.Run("Save_overwrites", func(t *testing.T) {
		const id = "sess-overwrite"
		first := SessionState{SessionID: id, ActiveGoal: "first"}
		updated := SessionState{SessionID: id, ActiveGoal: "updated"}

		if err := repo.Save(ctx, id, first); err != nil {
			t.Fatalf("Save first: %v", err)
		}
		if err := repo.Save(ctx, id, updated); err != nil {
			t.Fatalf("Save updated: %v", err)
		}

		got, err := repo.Load(ctx, id)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.ActiveGoal != updated.ActiveGoal {
			t.Errorf("ActiveGoal: got %q, want %q", got.ActiveGoal, updated.ActiveGoal)
		}
	})
}

func TestInMemorySessionRepository(t *testing.T) {
	repo := NewInMemorySessionRepository()
	sessionRepositorySuite(t, repo)
}

// newTestRedisClient 는 localhost:6379 Redis에 연결을 시도한다.
// Redis를 사용할 수 없으면 테스트를 건너뛴다.
func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("Redis unavailable (%v): skipping Redis tests", err)
	}
	return client
}

func TestRedisSessionRepository(t *testing.T) {
	client := newTestRedisClient(t)
	t.Cleanup(func() { _ = client.Close() })

	sessionRepositorySuite(t, NewRedisSessionRepository(client))
}

// TestRedisSessionRepository_PersistenceAfterReconnect 는 Redis 재시작 후 세션 복원을 검증한다.
// 새 클라이언트(새 연결)로 조회해도 이전에 저장한 데이터가 유지됨을 확인한다.
// 실제 Redis 재시작은 테스트에서 수행하지 않으며, 프로세스 재시작 시뮬레이션으로 새 클라이언트 생성을 사용한다.
// AOF persistence 활성화는 docker-compose.yml의 `--appendonly yes` 옵션으로 보장한다.
func TestRedisSessionRepository_PersistenceAfterReconnect(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-persist-test"

	client1 := newTestRedisClient(t)
	t.Cleanup(func() {
		client1.Del(ctx, sessionKeyPrefix+sessionID)
		_ = client1.Close()
	})

	want := SessionState{
		SessionID:   sessionID,
		ActiveGoal:  "persist goal",
		LastUpdated: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
	}

	repo1 := NewRedisSessionRepository(client1)
	if err := repo1.Save(ctx, sessionID, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 새 클라이언트로 재연결 — 프로세스 재시작 시뮬레이션
	client2 := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = client2.Close() })

	repo2 := NewRedisSessionRepository(client2)
	got, err := repo2.Load(ctx, sessionID)
	if err != nil {
		t.Fatalf("Load after reconnect: %v", err)
	}
	if got.SessionID != want.SessionID {
		t.Errorf("SessionID: got %q, want %q", got.SessionID, want.SessionID)
	}
	if got.ActiveGoal != want.ActiveGoal {
		t.Errorf("ActiveGoal: got %q, want %q", got.ActiveGoal, want.ActiveGoal)
	}
	if !got.LastUpdated.Equal(want.LastUpdated) {
		t.Errorf("LastUpdated: got %v, want %v", got.LastUpdated, want.LastUpdated)
	}
}
