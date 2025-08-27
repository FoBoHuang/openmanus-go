package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusSkipped    TaskStatus = "skipped"
)

// SubTask 子任务定义
type SubTask struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Type         string         `json:"type"` // "data_collection", "content_generation", "file_operation", "analysis", "other"
	Status       TaskStatus     `json:"status"`
	Priority     int            `json:"priority"`     // 1-10, 10最高
	Dependencies []string       `json:"dependencies"` // 依赖的子任务ID
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	Result       map[string]any `json:"result,omitempty"`   // 任务执行结果
	Evidence     []string       `json:"evidence,omitempty"` // 完成证据（如文件路径、API响应等）
}

// TaskPlan 任务计划
type TaskPlan struct {
	ID             string              `json:"id"`
	OriginalGoal   string              `json:"original_goal"`
	SubTasks       map[string]*SubTask `json:"sub_tasks"`
	ExecutionOrder []string            `json:"execution_order"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	Status         TaskStatus          `json:"status"`
}

// TaskManager 多步任务管理器
type TaskManager struct {
	llmClient   llm.Client
	currentPlan *TaskPlan
}

// NewTaskManager 创建任务管理器
func NewTaskManager(llmClient llm.Client) *TaskManager {
	return &TaskManager{
		llmClient: llmClient,
	}
}

// DecomposeGoal 分解目标为子任务
func (tm *TaskManager) DecomposeGoal(ctx context.Context, goal string) (*TaskPlan, error) {
	logger.Get().Sugar().Infow("task.decompose.start", "goal", goal)

	// 构建分解提示
	systemPrompt := tm.buildDecomposeSystemPrompt()
	userPrompt := tm.buildDecomposeUserPrompt(goal)

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	req := &llm.ChatRequest{
		Messages:    messages,
		Temperature: 0.1,
		MaxTokens:   3000,
	}

	response, err := tm.llmClient.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to decompose goal: %w", err)
	}

	// 获取响应内容
	var responseContent string
	if len(response.Choices) > 0 {
		responseContent = response.Choices[0].Message.Content
	} else {
		return nil, fmt.Errorf("no response choices received from LLM")
	}

	logger.Get().Sugar().Debugw("task.decompose.response", "content", responseContent)

	// 解析任务分解结果
	plan, err := tm.parseTaskPlan(goal, responseContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task plan: %w", err)
	}

	tm.currentPlan = plan
	logger.Get().Sugar().Infow("task.decompose.success",
		"plan_id", plan.ID,
		"subtask_count", len(plan.SubTasks),
		"execution_order", plan.ExecutionOrder)

	return plan, nil
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(taskID string, status TaskStatus, result map[string]any, evidence []string) error {
	if tm.currentPlan == nil {
		return fmt.Errorf("no active task plan")
	}

	task, exists := tm.currentPlan.SubTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	oldStatus := task.Status
	task.Status = status
	task.UpdatedAt = time.Now()

	if result != nil {
		task.Result = result
	}

	if evidence != nil {
		task.Evidence = evidence
	}

	if status == TaskStatusCompleted {
		now := time.Now()
		task.CompletedAt = &now
	}

	logger.Get().Sugar().Infow("task.status.updated",
		"task_id", taskID,
		"old_status", oldStatus,
		"new_status", status,
		"evidence_count", len(evidence))

	return nil
}

// GetNextTask 获取下一个应该执行的任务
func (tm *TaskManager) GetNextTask() *SubTask {
	if tm.currentPlan == nil {
		logger.Get().Sugar().Debugw("GetNextTask: no current plan")
		return nil
	}

	logger.Get().Sugar().Debugw("GetNextTask: checking tasks",
		"execution_order", tm.currentPlan.ExecutionOrder,
		"total_tasks", len(tm.currentPlan.SubTasks))

	// 按执行顺序查找第一个未完成的任务
	for i, taskID := range tm.currentPlan.ExecutionOrder {
		task, exists := tm.currentPlan.SubTasks[taskID]
		if !exists {
			logger.Get().Sugar().Warnw("GetNextTask: task not found", "task_id", taskID)
			continue
		}

		logger.Get().Sugar().Debugw("GetNextTask: checking task",
			"index", i,
			"task_id", taskID,
			"task_status", task.Status,
			"task_title", task.Title)

		if task.Status == TaskStatusPending {
			// 检查依赖是否已完成
			if tm.areDependenciesCompleted(task) {
				logger.Get().Sugar().Infow("GetNextTask: found next task",
					"task_id", taskID,
					"task_title", task.Title)
				return task
			} else {
				logger.Get().Sugar().Debugw("GetNextTask: dependencies not completed",
					"task_id", taskID,
					"dependencies", task.Dependencies)
			}
		}
	}

	logger.Get().Sugar().Debugw("GetNextTask: no available task found")
	return nil
}

// IsAllTasksCompleted 检查所有任务是否都已完成
func (tm *TaskManager) IsAllTasksCompleted() bool {
	if tm.currentPlan == nil {
		return false
	}

	for _, task := range tm.currentPlan.SubTasks {
		if task.Status != TaskStatusCompleted && task.Status != TaskStatusSkipped {
			return false
		}
	}

	return true
}

// GetCompletionSummary 获取完成情况摘要
func (tm *TaskManager) GetCompletionSummary() map[string]any {
	if tm.currentPlan == nil {
		return map[string]any{"error": "no active plan"}
	}

	completed := 0
	failed := 0
	skipped := 0
	pending := 0

	for _, task := range tm.currentPlan.SubTasks {
		switch task.Status {
		case TaskStatusCompleted:
			completed++
		case TaskStatusFailed:
			failed++
		case TaskStatusSkipped:
			skipped++
		case TaskStatusPending, TaskStatusInProgress:
			pending++
		}
	}

	total := len(tm.currentPlan.SubTasks)
	completionRate := float64(completed) / float64(total) * 100

	return map[string]any{
		"total_tasks":     total,
		"completed":       completed,
		"failed":          failed,
		"skipped":         skipped,
		"pending":         pending,
		"completion_rate": completionRate,
		"all_completed":   tm.IsAllTasksCompleted(),
		"original_goal":   tm.currentPlan.OriginalGoal,
	}
}

// areDependenciesCompleted 检查依赖任务是否已完成
func (tm *TaskManager) areDependenciesCompleted(task *SubTask) bool {
	for _, depID := range task.Dependencies {
		if depTask, exists := tm.currentPlan.SubTasks[depID]; exists {
			if depTask.Status != TaskStatusCompleted {
				return false
			}
		}
	}
	return true
}

// buildDecomposeSystemPrompt 构建任务分解系统提示
func (tm *TaskManager) buildDecomposeSystemPrompt() string {
	return `You are a Task Decomposition Expert. Your job is to break down complex goals into manageable subtasks.

Your analysis should:
1. **Identify Task Types**: Classify each subtask by type (data_collection, content_generation, file_operation, analysis, other)
2. **Set Dependencies**: Determine which tasks must be completed before others can start
3. **Assign Priorities**: Rate each task's importance (1-10, where 10 is highest priority)
4. **Create Execution Order**: Provide a logical sequence considering dependencies

Return your analysis as a JSON object with this exact format:
{
  "subtasks": [
    {
      "id": "task_1",
      "title": "Short descriptive title",
      "description": "Detailed description of what needs to be done",
      "type": "data_collection|content_generation|file_operation|analysis|other",
      "priority": 1-10,
      "dependencies": ["task_id1", "task_id2"]
    }
  ],
  "execution_order": ["task_1", "task_2", "task_3"]
}

CRITICAL RULES:
- Each subtask must have a unique ID (use snake_case: task_1, task_2, etc.)
- execution_order must list ALL subtask IDs in logical sequence
- Dependencies must reference existing task IDs
- For multi-step goals (like "analyze X AND save to file"), create separate subtasks
- File operations should typically come AFTER content generation
- Always create at least 2 subtasks for complex goals`
}

// buildDecomposeUserPrompt 构建任务分解用户提示
func (tm *TaskManager) buildDecomposeUserPrompt(goal string) string {
	return fmt.Sprintf(`**GOAL TO DECOMPOSE**: %s

Please analyze this goal and break it down into specific, actionable subtasks. Pay special attention to:

1. **Multiple Requirements**: If the goal contains words like "and", "then", "also", "并", "然后", it likely needs multiple subtasks
2. **Data Dependencies**: Information gathering must happen before analysis or file creation
3. **File Operations**: If the goal mentions saving, writing, or creating files, that's a separate subtask
4. **Content Generation**: Summarizing, analyzing, or generating content is typically a distinct subtask

Provide your decomposition in the specified JSON format.`, goal)
}

// parseTaskPlan 解析任务计划
func (tm *TaskManager) parseTaskPlan(originalGoal, responseContent string) (*TaskPlan, error) {
	// 尝试提取JSON内容
	jsonContent := tm.extractJSONFromResponse(responseContent)

	// 解析响应
	var decomposition struct {
		SubTasks       []SubTask `json:"subtasks"`
		ExecutionOrder []string  `json:"execution_order"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &decomposition); err != nil {
		return nil, fmt.Errorf("failed to parse decomposition JSON: %w", err)
	}

	// 验证数据完整性
	if len(decomposition.SubTasks) == 0 {
		return nil, fmt.Errorf("no subtasks found in decomposition")
	}

	if len(decomposition.ExecutionOrder) != len(decomposition.SubTasks) {
		return nil, fmt.Errorf("execution order length (%d) doesn't match subtasks count (%d)",
			len(decomposition.ExecutionOrder), len(decomposition.SubTasks))
	}

	// 创建任务计划
	plan := &TaskPlan{
		ID:             fmt.Sprintf("plan_%d", time.Now().Unix()),
		OriginalGoal:   originalGoal,
		SubTasks:       make(map[string]*SubTask),
		ExecutionOrder: decomposition.ExecutionOrder,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Status:         TaskStatusPending,
	}

	// 填充子任务
	for i, task := range decomposition.SubTasks {
		// 创建新的任务副本，避免引用问题
		newTask := SubTask{
			ID:           task.ID,
			Title:        task.Title,
			Description:  task.Description,
			Type:         task.Type,
			Priority:     task.Priority,
			Dependencies: make([]string, len(task.Dependencies)),
			Status:       TaskStatusPending,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// 复制依赖数组
		copy(newTask.Dependencies, task.Dependencies)

		plan.SubTasks[newTask.ID] = &newTask

		logger.Get().Sugar().Debugw("task.parse.added",
			"index", i,
			"task_id", newTask.ID,
			"task_title", newTask.Title,
			"task_type", newTask.Type,
			"dependencies", newTask.Dependencies)
	}

	// 验证执行顺序中的任务ID都存在
	for _, taskID := range plan.ExecutionOrder {
		if _, exists := plan.SubTasks[taskID]; !exists {
			return nil, fmt.Errorf("execution order references non-existent task: %s", taskID)
		}
	}

	return plan, nil
}

// extractJSONFromResponse 从响应中提取JSON内容
func (tm *TaskManager) extractJSONFromResponse(response string) string {
	// 尝试匹配 ```json...``` 代码块
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json")
		if start != -1 {
			start += 7 // 跳过 "```json"
			end := strings.Index(response[start:], "```")
			if end != -1 {
				return strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// 尝试匹配 ```...``` 代码块（无json标识）
	if strings.Count(response, "```") >= 2 {
		start := strings.Index(response, "```")
		if start != -1 {
			start += 3
			// 跳过可能的语言标识符
			if newlineIdx := strings.Index(response[start:], "\n"); newlineIdx != -1 {
				start += newlineIdx + 1
			}
			end := strings.Index(response[start:], "```")
			if end != -1 {
				content := strings.TrimSpace(response[start : start+end])
				// 检查内容是否看起来像JSON
				if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
					return content
				}
			}
		}
	}

	// 如果没有代码块，直接返回原内容
	return strings.TrimSpace(response)
}

// GetCurrentPlan 获取当前任务计划
func (tm *TaskManager) GetCurrentPlan() *TaskPlan {
	return tm.currentPlan
}

// DetectTaskCompletion 检测任务完成情况（基于执行轨迹）
func (tm *TaskManager) DetectTaskCompletion(trace *state.Trace) error {
	if tm.currentPlan == nil {
		return fmt.Errorf("no active task plan")
	}

	// 分析最近的步骤，自动更新任务状态
	if len(trace.Steps) > 0 {
		lastStep := trace.Steps[len(trace.Steps)-1]

		// 基于动作类型和结果自动更新任务状态
		if lastStep.Observation != nil && lastStep.Observation.ErrMsg == "" {
			// 成功的操作
			taskType := tm.inferTaskTypeFromAction(lastStep.Action.Name)
			evidence := tm.extractEvidenceFromObservation(lastStep.Observation)

			// 查找匹配的待完成任务
			for _, task := range tm.currentPlan.SubTasks {
				if task.Status == TaskStatusPending && task.Type == taskType {
					// 标记为完成
					result := map[string]any{
						"action": lastStep.Action.Name,
						"args":   lastStep.Action.Args,
					}
					tm.UpdateTaskStatus(task.ID, TaskStatusCompleted, result, evidence)
					logger.Get().Sugar().Infow("task.auto.completed",
						"task_id", task.ID,
						"task_type", taskType,
						"action", lastStep.Action.Name)
					break
				}
			}
		}
	}

	return nil
}

// inferTaskTypeFromAction 从动作推断任务类型
func (tm *TaskManager) inferTaskTypeFromAction(actionName string) string {
	switch actionName {
	case "crawler", "http", "http_client":
		return "data_collection"
	case "fs", "file_copy":
		return "file_operation"
	case "direct_answer":
		return "content_generation"
	default:
		return "other"
	}
}

// extractEvidenceFromObservation 从观测结果中提取证据
func (tm *TaskManager) extractEvidenceFromObservation(obs *state.Observation) []string {
	evidence := []string{}

	if obs.Output != nil {
		// obs.Output 已经是 map[string]any 类型
		if path, exists := obs.Output["path"]; exists {
			if pathStr, ok := path.(string); ok {
				evidence = append(evidence, fmt.Sprintf("file_created:%s", pathStr))
			}
		}

		// 添加操作成功的证据
		evidence = append(evidence, fmt.Sprintf("tool_success:%s", obs.Tool))
	}

	return evidence
}
