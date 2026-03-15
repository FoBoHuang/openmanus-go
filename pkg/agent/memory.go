package agent

import (
	"sync"
	"time"

	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"
)

const (
	shortTermDefaultTTL = 30 * time.Minute
)

// Memory Agent 记忆管理（短期记忆带 TTL + 长期记忆可持久化）
type Memory struct {
	currentTrace *state.Trace
	shortTerm    MemoryStore // 短期记忆：InMemoryStore，带 TTL 自动过期
	longTerm     MemoryStore // 长期记忆：FileStore 或 InMemoryStore，跨 session 持久化
	mu           sync.RWMutex
}

// MemoryConfig 记忆配置
type MemoryConfig struct {
	LongTermPath    string        // 长期记忆持久化文件路径，为空则使用内存存储
	ShortTermTTL    time.Duration // 短期记忆默认 TTL
}

// DefaultMemoryConfig 默认记忆配置
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		LongTermPath: "",
		ShortTermTTL: shortTermDefaultTTL,
	}
}

// NewMemory 创建记忆管理器（内存模式，兼容旧接口）
func NewMemory() *Memory {
	return NewMemoryWithConfig(DefaultMemoryConfig())
}

// NewMemoryWithConfig 根据配置创建记忆管理器
func NewMemoryWithConfig(cfg *MemoryConfig) *Memory {
	if cfg == nil {
		cfg = DefaultMemoryConfig()
	}

	shortTerm := NewInMemoryStore()

	var longTerm MemoryStore
	if cfg.LongTermPath != "" {
		fileStore, err := NewFileStore(cfg.LongTermPath)
		if err != nil {
			logger.Warnw("memory.long_term.file_store_failed, falling back to in-memory",
				"path", cfg.LongTermPath, "error", err)
			longTerm = NewInMemoryStore()
		} else {
			longTerm = fileStore
			logger.Infow("memory.long_term.file_store_ready", "path", cfg.LongTermPath)
		}
	} else {
		longTerm = NewInMemoryStore()
	}

	return &Memory{
		shortTerm: shortTerm,
		longTerm:  longTerm,
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

// SetShortTerm 设置短期记忆（带默认 TTL）
func (m *Memory) SetShortTerm(key string, value any) {
	m.shortTerm.Set(NewMemoryEntryWithTTL(key, value, "short_term", 0.5, shortTermDefaultTTL))
}

// SetShortTermWithTTL 设置带自定义 TTL 的短期记忆
func (m *Memory) SetShortTermWithTTL(key string, value any, ttl time.Duration) {
	m.shortTerm.Set(NewMemoryEntryWithTTL(key, value, "short_term", 0.5, ttl))
}

// GetShortTerm 获取短期记忆
func (m *Memory) GetShortTerm(key string) (any, bool) {
	entry, exists := m.shortTerm.Get(key)
	if !exists {
		return nil, false
	}
	return entry.Value, true
}

// SetLongTerm 设置长期记忆（自动持久化）
func (m *Memory) SetLongTerm(key string, value any) {
	m.SetLongTermWithImportance(key, value, "general", 0.5)
}

// SetLongTermWithImportance 设置带 importance 评分的长期记忆
func (m *Memory) SetLongTermWithImportance(key string, value any, category string, importance float64) {
	entry := NewMemoryEntry(key, value, category, importance)
	m.longTerm.Set(entry)
}

// GetLongTerm 获取长期记忆
func (m *Memory) GetLongTerm(key string) (any, bool) {
	entry, exists := m.longTerm.Get(key)
	if !exists {
		return nil, false
	}
	return entry.Value, true
}

// GetLongTermEntry 获取长期记忆条目（含元数据）
func (m *Memory) GetLongTermEntry(key string) (*MemoryEntry, bool) {
	return m.longTerm.Get(key)
}

// ClearShortTerm 清空短期记忆
func (m *Memory) ClearShortTerm() {
	entries := m.shortTerm.List()
	for _, entry := range entries {
		m.shortTerm.Delete(entry.Key)
	}
}

// CleanExpiredShortTerm 清理过期的短期记忆
func (m *Memory) CleanExpiredShortTerm() int {
	if store, ok := m.shortTerm.(*InMemoryStore); ok {
		return store.CleanExpired()
	}
	return 0
}

// FlushLongTerm 将长期记忆刷写到持久化存储
func (m *Memory) FlushLongTerm() error {
	return m.longTerm.Flush()
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

	summary["short_term_keys"] = len(m.shortTerm.List())
	summary["long_term_keys"] = len(m.longTerm.List())

	return summary
}

// CompressTrace 压缩轨迹以节省内存
func (m *Memory) CompressTrace(maxSteps int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentTrace == nil || len(m.currentTrace.Steps) <= maxSteps {
		return
	}

	recentSteps := m.currentTrace.Steps[len(m.currentTrace.Steps)-maxSteps:]
	compressedSummary := m.createCompressedSummary(m.currentTrace.Steps[:len(m.currentTrace.Steps)-maxSteps])

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

		if step.Summary != "" {
			if keyOutcomes, ok := summary["key_outcomes"].([]string); ok {
				if len(keyOutcomes) < 5 {
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

// AddContextualInfo 添加上下文信息（短期记忆）
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

	var successRate float64
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount)
	}

	m.currentTrace.Scratch["metrics"] = map[string]any{
		"success_count": successCount,
		"total_count":   totalCount,
		"success_rate":  successRate,
		"updated_at":    time.Now(),
	}
}
