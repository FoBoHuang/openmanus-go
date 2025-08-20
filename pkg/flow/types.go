package flow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/state"
)

// ExecutionMode 执行模式
type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential" // 顺序执行
	ExecutionModeParallel   ExecutionMode = "parallel"   // 并行执行
	ExecutionModeDAG        ExecutionMode = "dag"        // DAG 依赖执行
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 等待执行
	TaskStatusRunning   TaskStatus = "running"   // 正在执行
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败
	TaskStatusCanceled  TaskStatus = "canceled"  // 已取消
	TaskStatusSkipped   TaskStatus = "skipped"   // 已跳过
)

// FlowStatus 流程状态
type FlowStatus string

const (
	FlowStatusPending   FlowStatus = "pending"   // 等待开始
	FlowStatusRunning   FlowStatus = "running"   // 正在执行
	FlowStatusCompleted FlowStatus = "completed" // 已完成
	FlowStatusFailed    FlowStatus = "failed"    // 执行失败
	FlowStatusCanceled  FlowStatus = "canceled"  // 已取消
)

// Task 任务定义
type Task struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	AgentType    string                 `json:"agent_type"`   // Agent 类型
	Goal         string                 `json:"goal"`         // 任务目标
	Input        map[string]interface{} `json:"input"`        // 输入数据
	Output       map[string]interface{} `json:"output"`       // 输出数据
	Dependencies []string               `json:"dependencies"` // 依赖的任务 ID
	Status       TaskStatus             `json:"status"`
	Error        string                 `json:"error,omitempty"`
	StartTime    *time.Time             `json:"start_time,omitempty"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Trace        *state.Trace           `json:"trace,omitempty"` // 执行轨迹
}

// Workflow 工作流定义
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Tasks       []*Task                `json:"tasks"`
	Mode        ExecutionMode          `json:"mode"`
	Config      map[string]interface{} `json:"config,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// FlowExecution 流程执行实例
type FlowExecution struct {
	ID        string                 `json:"id"`
	Workflow  *Workflow              `json:"workflow"`
	Status    FlowStatus             `json:"status"`
	Input     map[string]interface{} `json:"input"`   // 流程输入
	Output    map[string]interface{} `json:"output"`  // 流程输出
	Context   map[string]interface{} `json:"context"` // 共享上下文
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`

	// 运行时状态
	mu           sync.RWMutex
	taskResults  map[string]interface{} // 任务结果缓存
	agentPool    map[string]agent.Agent // Agent 池
	cancelFunc   context.CancelFunc     // 取消函数
	eventChannel chan *FlowEvent        // 事件通道
}

// FlowEvent 流程事件
type FlowEvent struct {
	Type      FlowEventType          `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	FlowID    string                 `json:"flow_id"`
	TaskID    string                 `json:"task_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Message   string                 `json:"message,omitempty"`
}

// FlowEventType 流程事件类型
type FlowEventType string

const (
	FlowEventTypeFlowStarted   FlowEventType = "flow_started"
	FlowEventTypeFlowCompleted FlowEventType = "flow_completed"
	FlowEventTypeFlowFailed    FlowEventType = "flow_failed"
	FlowEventTypeFlowCanceled  FlowEventType = "flow_canceled"
	FlowEventTypeTaskStarted   FlowEventType = "task_started"
	FlowEventTypeTaskCompleted FlowEventType = "task_completed"
	FlowEventTypeTaskFailed    FlowEventType = "task_failed"
	FlowEventTypeTaskSkipped   FlowEventType = "task_skipped"
)

// AgentFactory Agent 工厂接口
type AgentFactory interface {
	CreateAgent(agentType string, config map[string]interface{}) (agent.Agent, error)
	GetSupportedTypes() []string
	ValidateAgentConfig(agentType string, config map[string]interface{}) error
}

// FlowEngine 流程引擎接口
type FlowEngine interface {
	// 执行工作流
	Execute(ctx context.Context, workflow *Workflow, input map[string]interface{}) (*FlowExecution, error)

	// 获取执行状态
	GetExecution(id string) (*FlowExecution, error)

	// 取消执行
	CancelExecution(id string) error

	// 监听事件
	Subscribe(executionID string) (<-chan *FlowEvent, error)

	// 清理资源
	Cleanup(executionID string) error
}

// TaskExecutor 任务执行器接口
type TaskExecutor interface {
	Execute(ctx context.Context, task *Task, execution *FlowExecution) error
}

// DependencyResolver 依赖解析器接口
type DependencyResolver interface {
	Resolve(tasks []*Task) ([][]*Task, error) // 返回执行层级
	Validate(tasks []*Task) error             // 验证依赖关系
}

// NewTask 创建新任务
func NewTask(id, name, agentType, goal string) *Task {
	return &Task{
		ID:           id,
		Name:         name,
		AgentType:    agentType,
		Goal:         goal,
		Input:        make(map[string]interface{}),
		Output:       make(map[string]interface{}),
		Dependencies: make([]string, 0),
		Status:       TaskStatusPending,
	}
}

// NewWorkflow 创建新工作流
func NewWorkflow(id, name string, mode ExecutionMode) *Workflow {
	now := time.Now()
	return &Workflow{
		ID:        id,
		Name:      name,
		Tasks:     make([]*Task, 0),
		Mode:      mode,
		Config:    make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewFlowExecution 创建新的流程执行实例
func NewFlowExecution(id string, workflow *Workflow) *FlowExecution {
	return &FlowExecution{
		ID:           id,
		Workflow:     workflow,
		Status:       FlowStatusPending,
		Input:        make(map[string]interface{}),
		Output:       make(map[string]interface{}),
		Context:      make(map[string]interface{}),
		taskResults:  make(map[string]interface{}),
		agentPool:    make(map[string]agent.Agent),
		eventChannel: make(chan *FlowEvent, 100),
	}
}

// AddTask 添加任务到工作流
func (w *Workflow) AddTask(task *Task) {
	w.Tasks = append(w.Tasks, task)
	w.UpdatedAt = time.Now()
}

// GetTask 根据 ID 获取任务
func (w *Workflow) GetTask(id string) *Task {
	for _, task := range w.Tasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

// Validate 验证工作流
func (w *Workflow) Validate() error {
	if len(w.Tasks) == 0 {
		return fmt.Errorf("workflow must have at least one task")
	}

	// 检查任务 ID 唯一性
	taskIDs := make(map[string]bool)
	for _, task := range w.Tasks {
		if taskIDs[task.ID] {
			return fmt.Errorf("duplicate task ID: %s", task.ID)
		}
		taskIDs[task.ID] = true
	}

	// 检查依赖关系
	for _, task := range w.Tasks {
		for _, depID := range task.Dependencies {
			if !taskIDs[depID] {
				return fmt.Errorf("task %s depends on non-existent task %s", task.ID, depID)
			}
		}
	}

	return nil
}

// IsCompleted 检查任务是否完成
func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted
}

// IsFailed 检查任务是否失败
func (t *Task) IsFailed() bool {
	return t.Status == TaskStatusFailed
}

// CanExecute 检查任务是否可以执行
func (t *Task) CanExecute(execution *FlowExecution) bool {
	if t.Status != TaskStatusPending {
		return false
	}

	// 检查依赖是否都已完成
	for _, depID := range t.Dependencies {
		depTask := execution.Workflow.GetTask(depID)
		if depTask == nil || !depTask.IsCompleted() {
			return false
		}
	}

	return true
}

// Start 开始执行任务
func (t *Task) Start() {
	t.Status = TaskStatusRunning
	now := time.Now()
	t.StartTime = &now
}

// Complete 完成任务
func (t *Task) Complete(output map[string]interface{}) {
	t.Status = TaskStatusCompleted
	t.Output = output
	now := time.Now()
	t.EndTime = &now
	if t.StartTime != nil {
		t.Duration = now.Sub(*t.StartTime)
	}
}

// Fail 任务失败
func (t *Task) Fail(err error) {
	t.Status = TaskStatusFailed
	t.Error = err.Error()
	now := time.Now()
	t.EndTime = &now
	if t.StartTime != nil {
		t.Duration = now.Sub(*t.StartTime)
	}
}

// Cancel 取消任务
func (t *Task) Cancel() {
	t.Status = TaskStatusCanceled
	now := time.Now()
	t.EndTime = &now
	if t.StartTime != nil {
		t.Duration = now.Sub(*t.StartTime)
	}
}

// Skip 跳过任务
func (t *Task) Skip(reason string) {
	t.Status = TaskStatusSkipped
	t.Error = reason
	now := time.Now()
	t.EndTime = &now
	if t.StartTime != nil {
		t.Duration = now.Sub(*t.StartTime)
	}
}

// GetTaskResult 获取任务结果
func (e *FlowExecution) GetTaskResult(taskID string) (interface{}, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result, exists := e.taskResults[taskID]
	return result, exists
}

// SetTaskResult 设置任务结果
func (e *FlowExecution) SetTaskResult(taskID string, result interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.taskResults[taskID] = result
}

// GetAgent 获取 Agent
func (e *FlowExecution) GetAgent(agentType string) (agent.Agent, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	ag, exists := e.agentPool[agentType]
	return ag, exists
}

// SetAgent 设置 Agent
func (e *FlowExecution) SetAgent(agentType string, ag agent.Agent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.agentPool[agentType] = ag
}

// EmitEvent 发送事件
func (e *FlowExecution) EmitEvent(eventType FlowEventType, taskID string, data map[string]interface{}, message string) {
	event := &FlowEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		FlowID:    e.ID,
		TaskID:    taskID,
		Data:      data,
		Message:   message,
	}

	select {
	case e.eventChannel <- event:
	default:
		// 事件通道满了，丢弃事件
	}
}

// GetEventChannel 获取事件通道
func (e *FlowExecution) GetEventChannel() <-chan *FlowEvent {
	return e.eventChannel
}

// Start 开始执行流程
func (e *FlowExecution) Start() {
	e.Status = FlowStatusRunning
	now := time.Now()
	e.StartTime = &now
	e.EmitEvent(FlowEventTypeFlowStarted, "", nil, "Flow execution started")
}

// Complete 完成流程
func (e *FlowExecution) Complete(output map[string]interface{}) {
	e.Status = FlowStatusCompleted
	e.Output = output
	now := time.Now()
	e.EndTime = &now
	if e.StartTime != nil {
		e.Duration = now.Sub(*e.StartTime)
	}
	e.EmitEvent(FlowEventTypeFlowCompleted, "", output, "Flow execution completed")
}

// Fail 流程失败
func (e *FlowExecution) Fail(err error) {
	e.Status = FlowStatusFailed
	e.Error = err.Error()
	now := time.Now()
	e.EndTime = &now
	if e.StartTime != nil {
		e.Duration = now.Sub(*e.StartTime)
	}
	e.EmitEvent(FlowEventTypeFlowFailed, "", map[string]interface{}{"error": err.Error()}, "Flow execution failed")
}

// Cancel 取消流程
func (e *FlowExecution) Cancel() {
	e.Status = FlowStatusCanceled
	now := time.Now()
	e.EndTime = &now
	if e.StartTime != nil {
		e.Duration = now.Sub(*e.StartTime)
	}
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
	e.EmitEvent(FlowEventTypeFlowCanceled, "", nil, "Flow execution canceled")
}

// SetCancelFunc 设置取消函数
func (e *FlowExecution) SetCancelFunc(cancelFunc context.CancelFunc) {
	e.cancelFunc = cancelFunc
}
