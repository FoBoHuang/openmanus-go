package state

import (
	"encoding/json"
	"time"
)

// Action 表示一个可执行的动作
type Action struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args"`
	Reason string         `json:"reason,omitempty"`
}

// Observation 表示工具执行的观测结果
type Observation struct {
	Tool    string         `json:"tool"`
	Output  map[string]any `json:"output"`
	ErrMsg  string         `json:"err_msg,omitempty"`
	Latency int64          `json:"latency_ms"`
}

// Step 表示执行轨迹中的一个步骤
type Step struct {
	Index       int          `json:"index"`
	Action      Action       `json:"action"`
	Observation *Observation `json:"observation,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Timestamp   time.Time    `json:"timestamp"`
}

// Trace 表示完整的执行轨迹
type Trace struct {
	Goal        string             `json:"goal"`
	Steps       []Step             `json:"steps"`
	Reflections []ReflectionRecord `json:"reflections,omitempty"` // 反思记录历史
	Scratch     map[string]any     `json:"scratch,omitempty"`
	Budget      Budget             `json:"budget"`
	Status      TraceStatus        `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// Budget 表示执行预算限制
type Budget struct {
	MaxSteps    int           `json:"max_steps"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	MaxDuration time.Duration `json:"max_duration,omitempty"`
	UsedSteps   int           `json:"used_steps"`
	UsedTokens  int           `json:"used_tokens"`
	StartTime   time.Time     `json:"start_time"`
}

// TraceStatus 表示轨迹状态
type TraceStatus string

const (
	TraceStatusRunning   TraceStatus = "running"
	TraceStatusCompleted TraceStatus = "completed"
	TraceStatusFailed    TraceStatus = "failed"
	TraceStatusCanceled  TraceStatus = "canceled"
)

// ReflectionResult 表示反思结果
type ReflectionResult struct {
	RevisePlan     bool    `json:"revise_plan"`
	NextActionHint string  `json:"next_action_hint"`
	ShouldStop     bool    `json:"should_stop"`
	Reason         string  `json:"reason"`
	Confidence     float64 `json:"confidence,omitempty"`
}

// ReflectionRecord 表示反思记录
type ReflectionRecord struct {
	StepIndex int              `json:"step_index"` // 进行反思时的步骤索引
	Result    ReflectionResult `json:"result"`     // 反思结果
	Timestamp time.Time        `json:"timestamp"`  // 反思时间
}

// DecisionType 表示决策类型
type DecisionType string

const (
	DecisionDirectAnswer     DecisionType = "DIRECT_ANSWER"
	DecisionUseTool          DecisionType = "USE_TOOL"
	DecisionAskClarification DecisionType = "ASK_CLARIFICATION"
	DecisionStop             DecisionType = "STOP"
)

// Decision 表示 Agent 的决策
type Decision struct {
	Type    DecisionType `json:"type"`
	Content string       `json:"content,omitempty"`
	Action  *Action      `json:"action,omitempty"`
	Reason  string       `json:"reason,omitempty"`
}

// ToJSON 将对象转换为 JSON 字符串
func (t *Trace) ToJSON() (string, error) {
	data, err := json.MarshalIndent(t, "", "  ")
	return string(data), err
}

// AddStep 添加新的执行步骤
func (t *Trace) AddStep(action Action) *Step {
	step := Step{
		Index:     len(t.Steps),
		Action:    action,
		Timestamp: time.Now(),
	}
	t.Steps = append(t.Steps, step)
	t.Budget.UsedSteps++
	t.UpdatedAt = time.Now()
	return &t.Steps[len(t.Steps)-1]
}

// UpdateObservation 更新最后一个步骤的观测结果
func (t *Trace) UpdateObservation(obs *Observation) {
	if len(t.Steps) > 0 {
		t.Steps[len(t.Steps)-1].Observation = obs
		t.UpdatedAt = time.Now()
	}
}

// AddReflection 添加反思记录
func (t *Trace) AddReflection(result *ReflectionResult) {
	reflection := ReflectionRecord{
		StepIndex: len(t.Steps),
		Result:    *result,
		Timestamp: time.Now(),
	}
	t.Reflections = append(t.Reflections, reflection)
	t.UpdatedAt = time.Now()
}

// GetLatestReflection 获取最新的反思记录
func (t *Trace) GetLatestReflection() *ReflectionRecord {
	if len(t.Reflections) == 0 {
		return nil
	}
	return &t.Reflections[len(t.Reflections)-1]
}

// IsExceededBudget 检查是否超出预算限制
func (t *Trace) IsExceededBudget() bool {
	if t.Budget.MaxSteps > 0 && t.Budget.UsedSteps >= t.Budget.MaxSteps {
		return true
	}
	if t.Budget.MaxTokens > 0 && t.Budget.UsedTokens >= t.Budget.MaxTokens {
		return true
	}
	if t.Budget.MaxDuration > 0 && time.Since(t.Budget.StartTime) >= t.Budget.MaxDuration {
		return true
	}
	return false
}
