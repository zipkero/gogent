# Tool Spec 문서

Phase 3에서 LLM system prompt에 tool spec을 삽입할 때 이 문서가 기준이 된다.
각 tool의 name, description, 입력 스키마, 출력 형식, 에러 케이스를 정의한다.

---

## 공통 규칙

- **입력**: `map[string]any` 형태로 전달된다. key는 각 tool의 입력 스키마 `FieldSchema.Name`과 일치해야 한다.
- **출력**: `types.ToolResult` 구조체로 반환된다 (`internal/types` 패키지).
  - `ToolName string` — tool 식별자
  - `Output string` — 정상 결과 문자열
  - `IsError bool` — 에러 발생 여부
  - `ErrMsg string` — 에러 메시지 (IsError=true일 때만 유효)
- **에러 구분**: `Execute()`가 `error`를 반환하면 `tool_execution_failed`(retryable). `ToolResult.IsError=true`는 실행은 성공했지만 입력 값이 유효하지 않거나 데이터가 없는 경우다 (fatal).
- **input validation**: ToolRouter가 `Execute()` 호출 전에 `InputSchema()`를 기반으로 required 필드 존재 여부와 타입을 검증한다. 실패 시 `input_validation_failed`(fatal) 에러를 반환한다.

---

## calculator

**패키지**: `internal/tools/calculator`

### 개요

수식 문자열을 받아 계산 결과를 반환한다. 사칙연산(`+`, `-`, `*`, `/`)과 괄호를 지원한다.
재귀하강 파서(recursive descent parser)로 구현되어 연산자 우선순위와 괄호 중첩을 정확히 처리한다.

### 입력 스키마

| 필드명       | 타입     | 필수 | 설명                                  |
|------------|--------|------|-------------------------------------|
| expression | string | yes  | 계산할 수식 문자열 (예: `'3 + 4 * (2 - 1)'`) |

### 출력 형식

```
"7"          // 정수처럼 표현 가능한 경우
"3.14"       // 소수점 결과
```

`strconv.FormatFloat`로 직렬화되며, 불필요한 trailing zero는 제거된다.

### 에러 케이스

| 조건                         | IsError | ErrMsg 예시                          |
|---------------------------|---------|-------------------------------------|
| `expression` 필드 누락 (router 이전) | — | `input_validation_failed` (ToolRouter에서 처리) |
| 0으로 나누기                  | true    | `"0으로 나눌 수 없습니다"`              |
| 잘못된 수식 (예: `"3 +"`)       | true    | `"예상치 못한 수식 끝"`                |
| 예상치 못한 문자 포함            | true    | `"예상치 못한 문자: \"abc\""` |
| 괄호 미닫힘                   | true    | `"')' 가 없습니다"`                    |

---

## weather_mock

**패키지**: `internal/tools/weather_mock`

### 개요

도시 이름을 받아 고정된 날씨 정보를 반환한다. 실제 API 대신 hardcoded mock 데이터를 사용한다.
도시명은 대소문자 및 공백을 무시하고 매칭한다.

### 입력 스키마

| 필드명 | 타입     | 필수 | 설명                               |
|------|--------|------|----------------------------------|
| city | string | yes  | 날씨를 조회할 도시 이름 (예: `'Seoul'`, `'Tokyo'`) |

### 지원 도시 목록

| 도시 key    | 한글명  | 날씨   | 기온(°C) | 습도(%) |
|-----------|-------|------|--------|-------|
| seoul     | 서울   | 맑음   | 18     | 45    |
| busan     | 부산   | 흐림   | 20     | 60    |
| jeju      | 제주   | 비    | 16     | 80    |
| incheon   | 인천   | 맑음   | 17     | 50    |
| daejeon   | 대전   | 안개   | 15     | 75    |
| tokyo     | 도쿄   | 맑음   | 22     | 55    |
| newyork   | 뉴욕   | 흐림   | 12     | 65    |
| london    | 런던   | 비    | 10     | 85    |
| paris     | 파리   | 맑음   | 19     | 48    |
| shanghai  | 상하이 | 흐림   | 25     | 70    |

입력 city를 소문자로 변환하고 공백을 제거한 후 위 key와 비교한다.

### 출력 형식

```
"도시: Seoul | 날씨: 맑음 | 기온: 18°C | 습도: 45%"
```

### 에러 케이스

| 조건                       | IsError | ErrMsg 예시                                    |
|--------------------------|---------|----------------------------------------------|
| `city` 필드 누락 (router 이전) | — | `input_validation_failed` (ToolRouter에서 처리) |
| 지원하지 않는 도시              | true    | `"'Berlin' 에 대한 날씨 데이터가 없습니다"`        |

---

## search_mock

**패키지**: `internal/tools/search_mock`

### 개요

쿼리 문자열을 받아 고정된 검색 결과를 반환한다. 쿼리를 소문자로 정규화한 후 keyword 포함 여부로 매칭한다.
하나의 쿼리에 여러 keyword가 매칭되면 모든 결과를 합쳐서 반환한다.

### 입력 스키마

| 필드명  | 타입     | 필수 | 설명                                       |
|-------|--------|------|------------------------------------------|
| query | string | yes  | 검색할 쿼리 문자열 (예: `'golang'`, `'AI agent'`) |

### 지원 키워드 및 결과 수

| keyword    | 결과 수 |
|-----------|-------|
| golang    | 2     |
| agent     | 2     |
| weather   | 2     |
| calculator | 1    |

### 출력 형식

```
[1] The Go Programming Language
    Go is an open source programming language...
    https://go.dev
[2] Go Documentation
    Official documentation for the Go programming language...
    https://go.dev/doc
```

각 결과는 `[순번] 제목 \n    스니펫 \n    URL` 형태로 줄 구분된다.

### 에러 케이스

| 조건                       | IsError | ErrMsg 예시                                    |
|--------------------------|---------|----------------------------------------------|
| `query` 필드 누락 (router 이전) | — | `input_validation_failed` (ToolRouter에서 처리) |
| 매칭되는 keyword 없음          | true    | `"'blockchain' 에 대한 검색 결과가 없습니다"` |
