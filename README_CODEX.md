# AI Runtime Curriculum

## 소개

이 문서는 LangChain, LangGraph 같은 고수준 프레임워크 없이 AI Runtime의 핵심 구조를 직접 구현하며 익히기 위한 커리큘럼이다.

목표는 단순한 LLM API 호출 앱을 만드는 것이 아니다.  
planner, executor, tool router, state, memory, verifier, scheduler 같은 runtime 구성요소를 직접 나누고 연결하면서, agent system이 실제로 어떻게 굴러가는지 몸으로 이해하는 데 있다.

이 커리큘럼은 "프레임워크 사용자"가 아니라 "runtime 설계자" 관점으로 사고하는 연습을 목표로 한다.

## 왜 프레임워크를 쓰지 않는가

LangChain이나 LangGraph 같은 프레임워크는 빠르게 데모를 만드는 데는 도움이 될 수 있다. 하지만 아래 같은 핵심 지점을 학습할 때는 오히려 내부 구조가 가려지기 쉽다.

- agent loop는 어떤 조건에서 반복되고 종료되는가
- planner는 어떤 형식의 결정을 내려야 하는가
- tool router는 planner 결과를 어떻게 실제 실행으로 연결하는가
- session state와 memory는 어떻게 구분해야 하는가
- 실패, retry, timeout, verification은 어디서 처리해야 하는가
- runtime을 나중에 서비스 구조로 확장하려면 어떤 경계가 먼저 필요할까

이 문서는 그런 질문에 직접 답할 수 있도록, 고수준 프레임워크 없이 작은 runtime을 단계적으로 구현하는 흐름을 제안한다.

## 학습 목표

이 커리큘럼을 따라가면 최종적으로 아래를 설명하고 구현할 수 있어야 한다.

- 최소 agent loop를 직접 구현할 수 있다.
- planner, executor, tool router의 책임을 분리할 수 있다.
- request state, session state, working memory, long-term memory를 구분할 수 있다.
- tool registry와 tool interface를 직접 설계할 수 있다.
- verifier, retry, timeout 같은 제어 흐름을 runtime에 녹일 수 있다.
- 단일 agent 구조를 multi-agent orchestration으로 확장할 수 있다.
- CLI 수준의 실행기를 API + worker 구조로 발전시키는 방향을 설명할 수 있다.

## 비목표

이 커리큘럼은 아래를 우선 목표로 두지 않는다.

- production-grade agent framework를 처음부터 완성하는 것
- 브라우저 자동화 중심 에이전트 만들기
- 자율 코딩 에이전트 만들기
- 특정 프레임워크 wrapper 만들기
- 화려한 데모를 빠르게 만드는 것

핵심은 "runtime 제어 흐름과 상태 모델을 제대로 이해하는 것"이다.

## 최종적으로 다루게 될 구성요소

이 문서가 다루는 runtime 핵심 구성요소는 아래와 같다.

- Agent
- Runtime
- Planner
- Executor
- Tool
- Tool Registry
- Tool Router
- Session
- Memory
- Verifier
- Scheduler
- Workflow
- Task
- Step

## 권장 기술 스택

### 언어

- Go

선정 이유:

- 인터페이스와 경계를 직접 설계하기 좋다.
- CLI, worker, API 서버, queue 기반 구조로 확장하기 좋다.
- runtime과 orchestration 중심 학습에 잘 맞는다.

### 1차 스택

- Go
- OpenAI 또는 Anthropic API 중 하나
- Redis
- Postgres
- Docker Compose

### 후반 단계 확장 스택

- pgvector 또는 Qdrant
- OpenTelemetry
- Prometheus / Grafana
- Kafka
- Kubernetes

## 구현 원칙

### 1. 처음에는 단일 프로세스로 시작한다

처음부터 마이크로서비스처럼 쪼개지 않는다.  
핵심 loop와 상태 모델을 이해하는 것이 먼저다.

### 2. 프레임워크보다 인터페이스를 먼저 만든다

라이브러리나 프레임워크 사용보다, planner / executor / tool / memory 같은 경계를 직접 정의하는 것을 우선한다.

### 3. 상태는 반드시 분리해서 다룬다

아래 상태를 하나로 뭉개지 않는다.

- request state
- session state
- working memory
- long-term memory

### 4. LLM은 핵심이 아니라 부품이다

이 커리큘럼의 중심은 model 호출 자체가 아니라 orchestration이다.

### 5. 각 단계마다 반드시 동작하는 결과물을 남긴다

문서만 쌓지 않는다.  
각 Phase마다 CLI 또는 API 기준으로 실제 동작하는 최소 산출물을 만든다.

## 전체 로드맵

```text
Phase 0  설계 준비 / 용어 정리
Phase 1  최소 Agent Loop
Phase 2  Tool Registry + Tool Router
Phase 3  Session / State / Memory 분리
Phase 4  Planner 고도화
Phase 5  Verifier / Reflection / Retry
Phase 6  Multi-Agent Orchestration
Phase 7  Runtime 서비스화
Phase 8  운영 관점 고도화
Phase 9  문서화 / 포트폴리오화
```

## Phase 0. 설계 준비 / 용어 정리

### 목적

이후 구현에서 흔들리지 않도록 핵심 용어와 경계를 먼저 고정한다.

### 해야 할 일

- `docs/glossary.md` 작성
- `docs/architecture-overview.md` 작성
- 이번 프로젝트의 범위와 제외 범위 명시

### 완료 기준

- Agent, Runtime, Planner, Executor, Tool, Memory 같은 핵심 용어 정의가 문서화되어 있다.
- 전체 흐름이 한 장의 문서로 요약되어 있다.
- 이번 커리큘럼에서 하지 않을 범위가 정리되어 있다.

## Phase 1. 최소 Agent Loop

### 목적

가장 작은 형태의 runtime loop를 직접 만든다.

```text
Input
 -> Plan
 -> Execute
 -> Observe
 -> Decide Finish
 -> Response
```

### 구성 목표

```text
cmd/agent-cli/
internal/agent/
internal/planner/
internal/executor/
internal/state/
internal/tools/
```

### 세부 단계

#### Step 1. CLI 입력기

- CLI로 사용자 입력을 받는다.
- request ID와 session ID를 만든다.
- 내부 runtime 진입점을 호출한다.

완료 기준:

- `go run ./cmd/agent-cli`로 입력을 받을 수 있다.
- 질문을 입력하면 runtime 진입점이 호출된다.

#### Step 2. AgentState 정의

예시 필드:

- RequestID
- SessionID
- UserInput
- CurrentPlan
- LastToolCall
- ToolResults
- FinalAnswer
- StepCount
- Status

완료 기준:

- loop의 각 단계에서 상태를 읽고 갱신할 수 있다.

#### Step 3. Planner 인터페이스

예시 역할:

- `Plan(state) -> PlanResult`

`PlanResult` 예시:

- next action type
- reason
- selected tool
- extracted params

완료 기준:

- mock planner로 최소 loop가 동작한다.

#### Step 4. Executor 인터페이스

예시 역할:

- planner 결과를 받아 tool 실행
- 결과를 상태에 반영
- 오류 기록

완료 기준:

- planner 결과가 executor를 통해 실제 실행 흐름으로 이어진다.

#### Step 5. 종료 조건

예시:

- max step 초과
- planner가 finish 판단
- 충분한 결과 도달

완료 기준:

- 질문 1개에 대해 loop가 정상 종료된다.

## Phase 2. Tool Registry + Tool Router

### 목적

tool을 하드코딩 호출하지 않고, 등록된 tool 목록에서 선택하고 실행할 수 있게 만든다.

### 세부 단계

#### Step 1. Tool 인터페이스 정의

예시:

- `Name()`
- `Description()`
- `InputSchema()`
- `Execute(ctx, input)`

#### Step 2. Tool Registry 구현

예시 tool:

- calculator
- weather
- search

완료 기준:

- tool 등록과 조회가 가능하다.
- registry 테스트가 존재한다.

#### Step 3. Tool Router 구현

planner 결과 예시:

```text
action=tool_call
tool=weather
params={city: seoul}
```

완료 기준:

- 올바른 tool로 연결된다.
- 미등록 tool과 잘못된 입력을 처리할 수 있다.

#### Step 4. Tool Spec 문서화

- `docs/tools.md`

완료 기준:

- 각 tool의 입력과 출력 형식이 문서에 정리되어 있다.

#### Step 5. Tool 실행 로그

남길 항목:

- request id
- session id
- tool name
- input
- output summary
- duration
- error 여부

## Phase 3. Session / State / Memory 분리

### 목적

대화 히스토리와 메모리를 뭉개지 않고, runtime에서 다뤄야 하는 상태를 구분한다.

### 분리 대상

1. Request State
2. Session State
3. Working Memory
4. Long-term Memory

### 세부 단계

#### Step 1. Request State

한 번의 실행에만 필요한 상태를 정의한다.

예:

- 이번 질문
- 이번 tool results
- 이번 reasoning steps

#### Step 2. Session State

같은 사용자의 연속 대화에서 이어지는 상태를 정의한다.

예:

- 최근 대화 맥락
- 현재 task context
- active goal

저장소 후보:

- 1차: in-memory map
- 2차: Redis

#### Step 3. Working Memory

문제 해결 과정 중간 산출물을 관리한다.

예:

- 검색 결과
- 필터링 결과
- 중간 요약

#### Step 4. Long-term Memory

오래 보관할 선호나 도메인 정보를 다룬다.

예:

- 사용자 선호
- 자주 쓰는 도시
- 선호 가격대

저장소 후보:

- Postgres

#### Step 5. Memory Manager

예시 기능:

- `LoadSession(sessionID)`
- `SaveSession(...)`
- `SavePreference(...)`
- `LoadRelevantMemory(...)`

## Phase 4. Planner 고도화

### 목적

mock planner를 넘어서 실제 LLM 기반 planner를 연결하고, 작업을 step 단위로 나눌 수 있게 만든다.

### 세부 단계

- action type 정의
- plan result schema 고정
- LLM planner 연결
- invalid JSON 대응
- simple task decomposition
- planner 테스트

예시 action type:

- finish
- ask_user
- tool_call
- summarize
- search_memory
- retry

## Phase 5. Verifier / Reflection / Retry

### 목적

실행 결과를 검증하고 필요하면 다시 시도하는 제어 흐름을 넣는다.

### 세부 단계

- verifier 인터페이스 정의
- retry 정책 정의
- reflection step 추가
- failure handling 분리

예시 failure type:

- tool not found
- malformed params
- timeout
- llm parse error
- empty result

## Phase 6. Multi-Agent Orchestration

### 목적

단일 agent loop를 manager / worker 구조로 확장한다.

### 예시 시나리오

- Search Agent
- Filter Agent
- Ranking Agent
- Summary Agent

### 세부 단계

- agent interface 분리
- manager agent 구현
- task contract 정의
- worker 결과 병합
- multi-agent trace 추가

## Phase 7. Runtime 서비스화

### 목적

CLI 수준 실행기를 API 서버 + worker 구조로 옮긴다.

### 세부 단계

- HTTP gateway 구현
- async task 모델 도입
- queue 도입
- worker 프로세스 분리
- admin / debug API 추가

예시 엔드포인트:

- `POST /v1/agent/run`
- `GET /v1/tasks/{id}`
- `GET /v1/sessions/{id}`

## Phase 8. 운영 관점 고도화

### 목적

runtime을 실제 운영 관점에서 바라보는 감각을 붙인다.

### 세부 단계

- timeout / cancellation
- 비용 추적
- observability
- OpenTelemetry
- 에러 분류 체계
- policy layer

추적 예시:

- request별 token 사용량
- planner 호출 수
- verifier 재시도 수
- tool latency

## Phase 9. 문서화 / 포트폴리오화

### 목적

이 커리큘럼 결과물을 설계 역량을 보여주는 자료로 정리한다.

### 세부 단계

- README 정리
- 아키텍처 문서 작성
- 실행 시나리오 문서화
- 개선 예정 항목 정리

문서 예시:

- `docs/01-runtime-overview.md`
- `docs/02-planner.md`
- `docs/03-memory.md`
- `docs/04-tool-router.md`
- `docs/05-multi-agent.md`

## 추천 진행 순서

### Sprint 1

- glossary 작성
- architecture overview 작성
- CLI 입력기
- AgentState 정의
- mock planner
- mock executor

### Sprint 2

- calculator tool
- weather mock tool
- tool registry
- tool router
- loop 종료 조건

### Sprint 3

- session state
- request state
- working memory
- Redis 연결
- session restore

### Sprint 4

- LLM planner 연결
- planner output schema 고정
- JSON parse validation
- planner 테스트

### Sprint 5

- verifier
- retry policy
- timeout handling
- failure classification

### Sprint 6

- search agent
- ranking agent
- summary agent
- manager agent
- multi-agent trace

### Sprint 7

- gateway API
- async task state
- worker 분리
- debug API

### Sprint 8

- observability
- token usage tracking
- OTel
- policy layer

### Sprint 9

- README 고도화
- docs 정리
- demo scenario 작성

## AI 도구 사용 원칙

Claude Code나 Codex 같은 도구는 아래처럼 역할을 나눠 쓰는 것이 좋다.

### Architect 역할

사용처:

- 디렉터리 구조 설계
- 인터페이스 설계
- 책임 분리
- 경계 정의

예시 요청:

- planner, executor, state 경계를 어떻게 나눌지 설계해줘
- tool registry 인터페이스 설계해줘

### Implementer 역할

사용처:

- Go 코드 작성
- 테스트 코드 작성
- handler 구현
- repository 구현

예시 요청:

- Go로 Tool 인터페이스와 registry 구현해줘
- Redis 기반 session repository 작성해줘

## 마무리

이 문서의 목적은 "AI agent 앱 하나 만들기"가 아니다.  
핵심은 runtime의 내부 구조를 추상화 뒤에 숨기지 않고 직접 구현해보면서, 나중에 어떤 프레임워크를 보더라도 그 내부 모델을 읽어낼 수 있는 감각을 만드는 데 있다.
