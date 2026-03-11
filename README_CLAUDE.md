# AI Agent Runtime

LangChain / LangGraph 같은 프레임워크 없이 AI Runtime을 직접 설계하고 구현하는 프로젝트.

프레임워크를 쓰지 않는 이유는 단순하다.
추상화가 너무 두꺼워서 loop가 어떻게 도는지, state가 어떻게 흘러가는지 보이지 않는다.
직접 만들어야 설계 감각이 생긴다.

---

## 목표

- Agent loop를 직접 구현할 수 있다
- Tool calling / Tool routing 구조를 설계할 수 있다
- Session state / Memory / Task state를 구분할 수 있다
- 단일 Agent에서 Multi-Agent로 확장할 수 있다
- Runtime을 서비스처럼 운영하는 구조를 설계할 수 있다

---

## 기술 스택

**언어:** Go

**1차 (Phase 0~6)**
- Anthropic API (Claude)
- Redis
- Postgres
- Docker Compose

**2차 (Phase 7~)**
- Kafka
- pgvector / Qdrant
- OpenTelemetry / Prometheus / Grafana
- Kubernetes

---

## 구현 원칙

1. **처음엔 단일 프로세스** — 처음부터 MSA로 쪼개지 않는다
2. **프레임워크보다 인터페이스** — 직접 인터페이스를 정의하고 구현한다
3. **상태를 항상 분리해서 본다** — request / session / memory / task
4. **LLM은 부품이다** — 핵심은 LLM이 아니라 orchestration이다
5. **단계마다 동작하는 결과물** — CLI 또는 API로 실제 동작해야 다음 단계로 간다

---

## 로드맵

```
Phase 0  준비
Phase 1  최소 Agent Loop
Phase 2  Tool Registry + Tool Router
Phase 3  Session / State / Memory 분리
Phase 4  Planner 고도화
Phase 5  Verifier / Reflection / Retry
Phase 6  Multi-Agent Orchestration
Phase 7  Runtime 서비스화
Phase 8  운영 고도화
Phase 9  문서화 / 포트폴리오
```

---

## Phase 0 — 준비

코드보다 개념을 먼저 고정하는 단계.
단, 문서만 쓰다 멈추지 않도록 프로젝트 초기화도 같이 한다.

### Step 0-1. LLM Provider 확정

**Anthropic API (Claude)** 로 확정한다.

Planner 인터페이스 설계 전에 provider를 고정해야 tool calling 스펙을 반영한 추상화를 제대로 만들 수 있다.
처음부터 LLM 추상화 레이어를 두어, 나중에 provider를 바꾸더라도 내부만 교체하면 되도록 설계한다.

```go
type LLMClient interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
```

### Step 0-2. 프로젝트 초기화

```
cmd/agent-cli/
internal/agent/
internal/planner/
internal/executor/
internal/state/
internal/tools/
docs/
```

- 각 디렉터리에 빈 인터페이스 stub 파일 생성
- `go build ./...` 통과 확인

### Step 0-3. 용어 정리

`docs/glossary.md` 작성.

Agent / Runtime / Planner / Executor / Tool / Tool Router / Session / Memory / Verifier / Task / Step 각각을 명확히 분리해서 정의한다.

### Step 0-4. 전체 흐름도

`docs/architecture-overview.md` 작성.

```
User Request → Runtime → Planner → Tool Router → Executor → Memory Update → Verifier → Response
```

### Step 0-5. 범위 고정

이번 프로젝트에서 **하지 않을 것**:
- 브라우저 자동조작
- 코드 수정형 에이전트
- 자율 배포 에이전트

**할 것**: QA / Search / Planning형으로 제한

### 완료 기준

- `go build ./...` 통과
- glossary, architecture-overview 문서 존재
- LLM Provider 확정
- 범위 문서 존재

---

## Phase 1 — 최소 Agent Loop

가장 먼저 만들 것은 loop의 뼈대다.

```
Input → Plan → Execute → Observe → Decide Finish → Response
```

### Step 1-1. CLI 입력기

```bash
go run ./cmd/agent-cli
> 오늘 서울 날씨 알려줘
```

- stdin 입력
- request ID 생성
- session ID 임시 고정
- 완료 기준: 입력 → `runtime.Run()` 호출

### Step 1-2. AgentState 구조

```go
type AgentState struct {
    RequestID   string
    SessionID   string
    UserInput   string
    CurrentPlan PlanResult
    ToolResults []ToolResult
    FinalAnswer string
    StepCount   int
    Status      AgentStatus
}
```

### Step 1-3. Planner 인터페이스

```go
type Planner interface {
    Plan(ctx context.Context, state AgentState) (PlanResult, error)
}
```

PlanResult 필드: action type / selected tool / extracted params / reason

action type은 3개로 제한한다:
```go
const (
    ActionToolCall       ActionType = "tool_call"
    ActionRespondDirectly ActionType = "respond_directly"
    ActionFinish         ActionType = "finish"
)
```

Mock planner로도 loop가 돌아야 한다. planner와 loop는 반드시 분리되어야 한다.

### Step 1-4. Executor 인터페이스

```go
type Executor interface {
    Execute(ctx context.Context, plan PlanResult) (ToolResult, error)
}
```

### Step 1-5. Finish 조건

- planner가 `finish` 반환
- max step 초과
- fatal error 발생
- `respond_directly` 완료

종료 시 반드시 종료 사유를 state에 기록한다.

### 테스트

- mock planner로 loop unit test
  - `tool_call` → 실행 → state 반영 흐름
  - `finish` 시 루프 종료
  - max step 초과 시 강제 종료
- planner를 교체해도 loop가 동작하는지 확인

### 완료 기준

- 질문 1개에 대해 루프가 정상 종료된다
- mock planner로도 동작한다
- 실행 로그에 step / action / tool / 종료 사유가 남는다

---

## Phase 2 — Tool Registry + Tool Router

하드코딩 호출 → 등록된 목록에서 선택 후 실행.

### Step 2-1. Tool 인터페이스

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() Schema
    Execute(ctx context.Context, input map[string]any) (ToolResult, error)
}
```

### Step 2-2. Tool Registry

초기 tool 목록: `calculator`, `weather_mock`, `search_mock`

- name으로 조회 가능
- 미등록 tool 조회 시 명확한 에러

### Step 2-3. Tool Router

planner 결과 → 실제 tool 연결.

처리해야 할 케이스:
- 미등록 tool name
- input validation 실패
- tool execute 에러

### Step 2-4. Tool 실행 로그

각 tool call마다 기록: request id / session id / tool name / input / output summary / duration / error 여부

### 테스트

- registry: 등록 → 조회 → 미등록 에러 unit test
- router: 유효 / 잘못된 tool name 라우팅 unit test
- 각 tool: 정상 입력 / 잘못된 입력 unit test

### 완료 기준

- 새 tool 추가 시 registry에 등록만 하면 된다
- planner와 tool이 느슨하게 결합된다

---

## Phase 3 — Session / State / Memory 분리

대화 히스토리 = 메모리로 뭉개는 게 흔한 실수다. 4가지를 명확히 분리한다.

| 종류 | 의미 | 생명주기 | 저장소 |
|---|---|---|---|
| Request State | 이번 실행 중 필요한 상태 | 요청 종료 시 폐기 | 메모리 |
| Session State | 연속 대화 맥락 | 세션 유지 | Redis |
| Working Memory | 문제 해결 중간 산출물 | 작업 종료 시 폐기 | 메모리 |
| Long-term Memory | 사용자/도메인 장기 정보 | 영구 | Postgres |

### Step 3-1~4. 각 상태 구조 정의

각 상태의 필드, 생명주기, 저장소를 코드로 표현한다.

### Step 3-5. Memory Manager

```go
type MemoryManager interface {
    LoadSession(sessionID string) (SessionState, error)
    SaveSession(sessionID string, state SessionState) error
    SavePreference(userID string, pref Preference) error
    LoadRelevantMemory(query string) ([]Memory, error)
}
```

runtime이 저장소를 직접 몰라도 된다.

### 테스트

- Session State: 동일 session ID로 맥락 이어지는지 unit test
- Memory Manager: in-memory 구현으로 저장 → 조회 unit test

### 완료 기준

- session ID 기준으로 대화 맥락이 이어진다
- state와 memory가 코드 레벨에서 분리된다

---

## Phase 4 — Planner 고도화

mock에서 실제 LLM planner로 교체한다.

### Step 4-1. Action 타입 확장

Phase 1의 3개에서 확장:

```go
const (
    ActionToolCall       ActionType = "tool_call"
    ActionRespondDirectly ActionType = "respond_directly"
    ActionFinish         ActionType = "finish"
    ActionAskUser        ActionType = "ask_user"
    ActionSummarize      ActionType = "summarize"
    ActionSearchMemory   ActionType = "search_memory"
    ActionRetry          ActionType = "retry"
)
```

### Step 4-2. PlanResult 스키마 고정

```go
type PlanResult struct {
    Action          ActionType
    ToolName        string
    ToolInput       map[string]any
    ReasoningSummary string
    Confidence      float64
    NextGoal        string
}
```

### Step 4-3. LLM Planner 연결

구현 포인트:
- system prompt 설계 (action schema 강제 포함)
- invalid JSON 대응
- hallucination 최소화

### Step 4-4. Token Usage 로깅

LLM을 연결하는 이 시점부터 token 사용량을 기록한다.
비용 추적을 나중으로 미루면 Phase 4부터 이미 쓴 token을 알 방법이 없다.

기록 항목:
- request id / prompt tokens / completion tokens / total tokens
- planner 호출 횟수 / session 누적 token

최소 구현은 로그 출력으로 충분하다. 정책은 Phase 8에서 추가한다.

### Step 4-5. Task Decomposition

"서울 저렴한 호텔 찾고 평점 높은 순으로 보여줘" → hotel search → filter → sort → summarize

### 테스트

- invalid JSON 반환 시 에러 처리 unit test
- PlanResult 스키마: 각 action type별 필수 필드 검증
- token logger: 호출마다 기록 여부 확인

### 완료 기준

- planner가 실제 LLM으로 tool 선택과 finish 판단을 한다
- LLM 호출마다 token 사용량이 로그에 남는다

---

## Phase 5 — Verifier / Reflection / Retry

실행 결과를 검증하고 필요하면 재시도한다. 여기서부터 Agent다운 느낌이 생긴다.

### Step 5-1. Verifier 인터페이스

```go
type Verifier interface {
    Verify(ctx context.Context, state AgentState) (VerifyResult, error)
}

// done / retry / fail 중 하나 반환
```

### Step 5-2. Retry 정책

- tool timeout → 1회 재시도
- planner invalid output → 재생성
- 빈 결과 → 다른 파라미터로 재시도

retry policy는 코드로 분리되어야 한다. loop 안에 인라인으로 쓰지 않는다.

### Step 5-3. Reflection

"현재 결과가 질문에 충분한가?", "누락된 조건이 있는가?" 같은 self-check.

### Step 5-4. Failure 분류

| 유형 | 처리 |
|---|---|
| tool not found | 에러 상태로 종료 |
| malformed params | planner 재호출 |
| timeout | retry policy 적용 |
| llm parse error | 재생성 시도 |
| empty result | 파라미터 변경 후 재시도 |

### 테스트

- Verifier: done / retry / fail 각 케이스 unit test
- Retry policy: 실패 유형별 재시도 횟수 제한 검증

### 완료 기준

- 실패 유형별 응답 정책이 코드로 분리되어 있다
- 무한 재시도가 발생하지 않는다

---

## Phase 6 — Multi-Agent Orchestration

단일 agent에서 Manager + Worker 구조로 확장한다.

### 시나리오 (호텔 도메인)

Search Agent → Filter Agent → Ranking Agent → Summary Agent

### Step 6-1. Agent 인터페이스

```go
type Agent interface {
    Name() string
    CanHandle(task Task) bool
    Execute(ctx context.Context, task Task) (TaskResult, error)
}
```

### Step 6-2. Task Contract

agent 간 주고받는 데이터 형식 통일:

```go
type Task struct {
    ID           string
    Type         string
    InputPayload map[string]any
    Dependencies []string
}
```

### Step 6-3. Manager Agent

- 요청 분석 → 필요한 worker 판단 → 순서 결정 → 결과 병합

### Step 6-4. Multi-Agent 실행 로그

기록: 호출된 agent / 호출 순서 / 각 latency / 실패 지점

### 테스트

- Manager: worker 선택 로직 unit test
- Task Contract: 결과 병합 검증

### 완료 기준

- manager → worker 흐름이 동작한다
- multi-agent trace를 로그로 볼 수 있다

---

## Phase 7 — Runtime 서비스화

CLI 장난감 → API 서버 + 백그라운드 워커.

### API

```
POST /v1/agent/run
GET  /v1/tasks/{id}
GET  /v1/sessions/{id}
```

### Async Task 상태

`queued → running → succeeded / failed`

### 구조

- 1차: in-memory queue
- 2차: Redis Stream 또는 Kafka
- API 서버와 runtime worker를 논리적으로 분리

### Admin / Debug API

최근 task 목록 / 실패 task 조회 / session dump / tool stats

### 테스트

- 각 엔드포인트 integration test
- 상태 전이 검증 (queued → running → succeeded)

### 완료 기준

- CLI 없이 API로 실행 가능
- task 상태를 조회할 수 있다

---

## Phase 8 — 운영 고도화

### Timeout / Cancellation

- tool별 timeout 설정
- 전체 request deadline 설정

### 비용 제어 (Phase 4 로깅 확장)

- session별 token 누적량 추적
- 비용 한도 초과 시 중단 정책

### Observability

- structured logging + trace ID
- tool / planner latency metrics
- OpenTelemetry: request → planner → tool → verifier trace 연결

### 에러 분류

user error / system error / provider error / tool error / retryable error / fatal error

### Policy Layer

- tool 사용 제한
- 사용자별 max step 제한
- 비용 한도 초과 시 중단

---

## Phase 9 — 문서화 / 포트폴리오

### README 고도화

- 프로젝트 소개
- 왜 프레임워크를 쓰지 않았는가
- 핵심 구조
- 실행 방법 + 예시 시나리오

### 아키텍처 문서

```
docs/01-runtime-overview.md
docs/02-planner.md
docs/03-memory.md
docs/04-tool-router.md
docs/05-multi-agent.md
```

### 실행 시나리오 문서

- 날씨 질의 흐름
- 호텔 검색 흐름
- 실패 후 retry 흐름
- multi-agent 흐름

---

## 단계별 산출물 체크리스트

매 Phase마다 아래 4개를 남긴다.

- [ ] 동작하는 코드
- [ ] 테스트
- [ ] docs 또는 README 갱신
- [ ] 다음 Phase TODO

이 4개 없이 다음 단계로 넘어가지 않는다.

---

## 하지 말아야 할 것

처음부터 하면 안 되는 것:
- 여러 LLM provider 동시 지원
- 처음부터 MSA 구조
- 처음부터 Kubernetes
- 브라우저 agent

먼저 loop → state → planner → tool router → verifier → session memory 순으로 엔진을 만든다.
엔진이 된 다음에 서비스화로 간다.
