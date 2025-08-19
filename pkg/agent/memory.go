package agent

import (
	"sync"
	"time"

	"openmanus-go/pkg/state"
)

// Memory Agent 记忆管理
type Memory struct {
	currentTrace *state.Trace
	shortTerm    map[string]any
	longTerm     map[string]any
	mu           sync.RWMutex
}

// NewMemory 创建记忆管理器
func NewMemory() *Memory {
	return &Memory{
		shortTerm: make(map[string]any),
		longTerm:  make(map[string]any),
	}
}

// SetCurrentTrace 设置当前轨迹
func (m *Memory) SetCurrentTrace(trace *state.Trace) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTrace = trace
}

// GetCurrentTrace 获取当前轨迹
func (m *Memory) GetCurrentTrace() *state.Trace {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentTrace
}

// SetShortTerm 设置短期记忆
func (m *Memory) SetShortTerm(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shortTerm[key] = value
}

// GetShortTerm 获取短期记忆
func (m *Memory) GetShortTerm(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.shortTerm[key]
	return value, exists
}

// SetLongTerm 设置长期记忆
func (m *Memory) SetLongTerm(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.longTerm[key] = value
}

// GetLongTerm 获取长期记忆
func (m *Memory) GetLongTerm(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.longTerm[key]
	return value, exists
}

// ClearShortTerm 清空短期记忆
func (m *Memory) ClearShortTerm() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shortTerm = make(map[string]any)
}

// GetSummary 获取记忆总结
func (m *Memory) GetSummary() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := make(map[string]any)

	if m.currentTrace != nil {
		summary["current_trace"] = map[string]any{
			"goal":       m.currentTrace.Goal,
			"steps":      len(m.currentTrace.Steps),
			"status":     m.currentTrace.Status,
			"created_at": m.currentTrace.CreatedAt,
			"updated_at": m.currentTrace.UpdatedAt,
		}
	}

	summary["short_term_keys"] = len(m.shortTerm)
	summary["long_term_keys"] = len(m.longTerm)

	return summary
}

// CompressTrace 压缩轨迹以节省内存
func (m *Memory) CompressTrace(maxSteps int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentTrace == nil || len(m.currentTrace.Steps) <= maxSteps {
		return
	}

	// 保留最近的步骤
	recentSteps := m.currentTrace.Steps[len(m.currentTrace.Steps)-maxSteps:]

	// 创建压缩的总结
	compressedSummary := m.createCompressedSummary(m.currentTrace.Steps[:len(m.currentTrace.Steps)-maxSteps])

	// 更新轨迹
	m.currentTrace.Steps = recentSteps
	if m.currentTrace.Scratch == nil {
		m.currentTrace.Scratch = make(map[string]any)
	}
	m.currentTrace.Scratch["compressed_history"] = compressedSummary
}

// createCompressedSummary 创建压缩的历史总结
func (m *Memory) createCompressedSummary(steps []state.Step) map[string]any {
	summary := map[string]any{
		"total_steps":   len(steps),
		"compressed_at": time.Now(),
		"action_counts": make(map[string]int),
		"success_rate":  0.0,
		"key_outcomes":  []string{},
	}

	actionCounts := make(map[string]int)
	successCount := 0

	for _, step := range steps {
		actionCounts[step.Action.Name]++

		if step.Observation != nil && step.Observation.ErrMsg == "" {
			successCount++
		}

		// 收集关键结果
		if step.Summary != "" {
			if keyOutcomes, ok := summary["key_outcomes"].([]string); ok {
				if len(keyOutcomes) < 5 { // 最多保留 5 个关键结果
					summary["key_outcomes"] = append(keyOutcomes, step.Summary)
				}
			}
		}
	}

	summary["action_counts"] = actionCounts
	if len(steps) > 0 {
		summary["success_rate"] = float64(successCount) / float64(len(steps))
	}

	return summary
}

// GetRecentSteps 获取最近的步骤
func (m *Memory) GetRecentSteps(count int) []state.Step {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentTrace == nil || len(m.currentTrace.Steps) == 0 {
		return nil
	}

	steps := m.currentTrace.Steps
	if len(steps) <= count {
		return steps
	}

	return steps[len(steps)-count:]
}

// GetFailedSteps 获取失败的步骤
func (m *Memory) GetFailedSteps() []state.Step {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentTrace == nil {
		return nil
	}

	var failedSteps []state.Step
	for _, step := range m.currentTrace.Steps {
		if step.Observation != nil && step.Observation.ErrMsg != "" {
			failedSteps = append(failedSteps, step)
		}
	}

	return failedSteps
}

// GetSuccessfulSteps 获取成功的步骤
func (m *Memory) GetSuccessfulSteps() []state.Step {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentTrace == nil {
		return nil
	}

	var successfulSteps []state.Step
	for _, step := range m.currentTrace.Steps {
		if step.Observation != nil && step.Observation.ErrMsg == "" {
			successfulSteps = append(successfulSteps, step)
		}
	}

	return successfulSteps
}

// AddContextualInfo 添加上下文信息
func (m *Memory) AddContextualInfo(key string, info any) {
	m.SetShortTerm("context_"+key, info)
}

// GetContextualInfo 获取上下文信息
func (m *Memory) GetContextualInfo(key string) (any, bool) {
	return m.GetShortTerm("context_" + key)
}

// UpdateTraceMetrics 更新轨迹指标
func (m *Memory) UpdateTraceMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentTrace == nil {
		return
	}

	// 计算成功率
	var successCount, totalCount int
	for _, step := range m.currentTrace.Steps {
		if step.Observation != nil {
			totalCount++
			if step.Observation.ErrMsg == "" {
				successCount++
			}
		}
	}

	if m.currentTrace.Scratch == nil {
		m.currentTrace.Scratch = make(map[string]any)
	}

	m.currentTrace.Scratch["metrics"] = map[string]any{
		"success_count": successCount,
		"total_count":   totalCount,
		"success_rate":  float64(successCount) / float64(totalCount),
		"updated_at":    time.Now(),
	}
}
