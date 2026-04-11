package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zipkero/agent-runtime/internal/types"
)

// PostgresMemoryRepository 는 Postgres 에 Memory 를 영구 저장하는 구현체다.
// 태그 조회는 `tags && $1` (배열 교집합) 연산자로 OR 조건을 구현하며
// GIN 인덱스(Migrate 에서 생성)를 통해 가속된다.
type PostgresMemoryRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresMemoryRepository 는 주어진 pgxpool.Pool 로 저장소를 생성한다.
// 테이블 및 인덱스는 Migrate 를 통해 미리 준비되어 있어야 한다.
func NewPostgresMemoryRepository(pool *pgxpool.Pool) *PostgresMemoryRepository {
	return &PostgresMemoryRepository{pool: pool}
}

// Save 는 Memory 레코드를 memories 테이블에 INSERT 한다.
// 호출측(MemoryManager)은 ID, UserID, Content, Tags, CreatedAt 을 모두 세팅한 상태로 넘겨야 한다.
func (r *PostgresMemoryRepository) Save(ctx context.Context, memory types.Memory) error {
	const query = `
INSERT INTO memories (id, user_id, content, tags, created_at)
VALUES ($1, $2, $3, $4, $5)`

	if _, err := r.pool.Exec(ctx, query, memory.ID, memory.UserID, memory.Content, memory.Tags, memory.CreatedAt); err != nil {
		return fmt.Errorf("memory: insert memory %s: %w", memory.ID, err)
	}
	return nil
}

// LoadByTags 는 tags 중 하나라도 일치하는 Memory 를 최신순으로 최대 limit 개 반환한다.
// tags 가 비어 있거나 limit 이 0 이하이면 빈 슬라이스를 반환한다.
func (r *PostgresMemoryRepository) LoadByTags(ctx context.Context, tags []string, limit int) ([]types.Memory, error) {
	if len(tags) == 0 || limit <= 0 {
		return []types.Memory{}, nil
	}

	const query = `
SELECT id, user_id, content, tags, created_at
FROM memories
WHERE tags && $1
ORDER BY created_at DESC
LIMIT $2`

	rows, err := r.pool.Query(ctx, query, tags, limit)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []types.Memory{}, nil
		}
		return nil, fmt.Errorf("memory: query memories by tags: %w", err)
	}
	defer rows.Close()

	result := make([]types.Memory, 0, limit)
	for rows.Next() {
		var m types.Memory
		if err := rows.Scan(&m.ID, &m.UserID, &m.Content, &m.Tags, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("memory: scan memory row: %w", err)
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate memory rows: %w", err)
	}
	return result, nil
}
