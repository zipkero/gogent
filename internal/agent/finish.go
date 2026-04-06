package agent

import (
	"agentflow/internal/state"
	"agentflow/internal/types"
)

const DefaultMaxStep = 10

// FinishReason 은 loop가 종료된 이유를 나타낸다.
type FinishReason string

const (
	// FinishByAction 은 Planner가 ActionFinish 를 반환해 loop가 종료된 경우다.
	FinishByAction FinishReason = "action_finish"
	// FinishByDirectResponse 는 Planner가 ActionRespondDirectly 를 반환하고
	// FinalAnswer 가 채워진 경우다.
	FinishByDirectResponse FinishReason = "direct_response"
	// FinishByMaxStep 은 StepCount 가 maxStep 에 도달해 loop가 강제 종료된 경우다.
	FinishByMaxStep FinishReason = "max_step"
	// FinishBySummarize 는 Planner가 ActionSummarize 를 반환해 loop가 종료된 경우다.
	FinishBySummarize FinishReason = "summarize"
	// FinishByFatalError 는 복구 불가능한 에러로 loop가 종료된 경우다.
	FinishByFatalError FinishReason = "fatal_error"
)

// FinishResult 는 IsFinished 의 반환값이다.
// Finished 가 true 일 때만 Reason 이 유효하다.
type FinishResult struct {
	Finished bool
	Reason   FinishReason
}

// IsFinished 는 현재 loop를 종료해야 하는지 판단한다.
// plan 은 이번 step에서 Planner 가 반환한 결정이다.
// s 는 plan 이 반영되기 전의 AgentState 다.
// maxStep 이 0 이하면 DefaultMaxStep 을 사용한다.
func IsFinished(plan types.PlanResult, s state.AgentState, maxStep int) FinishResult {
	if maxStep <= 0 {
		maxStep = DefaultMaxStep
	}

	// 1. Planner 가 명시적으로 finish 를 선택한 경우
	if plan.ActionType == types.ActionFinish {
		return FinishResult{Finished: true, Reason: FinishByAction}
	}

	// 2. respond_directly 이고 FinalAnswer 가 이미 채워진 경우
	if plan.ActionType == types.ActionRespondDirectly && s.FinalAnswer != "" {
		return FinishResult{Finished: true, Reason: FinishByDirectResponse}
	}

	// 2a. summarize 이고 FinalAnswer 가 이미 채워진 경우
	if plan.ActionType == types.ActionSummarize && s.FinalAnswer != "" {
		return FinishResult{Finished: true, Reason: FinishBySummarize}
	}

	// 3. StepCount 가 maxStep 에 도달한 경우
	if s.StepCount >= maxStep {
		return FinishResult{Finished: true, Reason: FinishByMaxStep}
	}

	// 4. Status 가 이미 종료 상태인 경우 (fatal error 등 외부에서 변경된 경우)
	if s.Status == state.StatusFailed {
		return FinishResult{Finished: true, Reason: FinishByFatalError}
	}

	return FinishResult{Finished: false}
}
