package flow

import (
	"fmt"
	"sort"
)

// DefaultDependencyResolver 默认依赖解析器
type DefaultDependencyResolver struct{}

// NewDefaultDependencyResolver 创建默认依赖解析器
func NewDefaultDependencyResolver() *DefaultDependencyResolver {
	return &DefaultDependencyResolver{}
}

// Resolve 解析任务依赖关系，返回执行层级
func (r *DefaultDependencyResolver) Resolve(tasks []*Task) ([][]*Task, error) {
	if err := r.Validate(tasks); err != nil {
		return nil, err
	}

	// 构建任务映射
	taskMap := make(map[string]*Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	// 计算每个任务的入度（依赖数量）
	inDegree := make(map[string]int)
	for _, task := range tasks {
		inDegree[task.ID] = len(task.Dependencies)
	}

	var levels [][]*Task
	remaining := make(map[string]*Task)
	for _, task := range tasks {
		remaining[task.ID] = task
	}

	// 拓扑排序，按层级分组
	for len(remaining) > 0 {
		var currentLevel []*Task

		// 找到当前层级可执行的任务（入度为0）
		for taskID, task := range remaining {
			if inDegree[taskID] == 0 {
				currentLevel = append(currentLevel, task)
			}
		}

		if len(currentLevel) == 0 {
			// 存在循环依赖
			return nil, fmt.Errorf("circular dependency detected among tasks: %v", getTaskIDs(remaining))
		}

		// 按任务 ID 排序，确保执行顺序的确定性
		sort.Slice(currentLevel, func(i, j int) bool {
			return currentLevel[i].ID < currentLevel[j].ID
		})

		levels = append(levels, currentLevel)

		// 移除当前层级的任务，并更新依赖它们的任务的入度
		for _, task := range currentLevel {
			delete(remaining, task.ID)

			// 更新依赖当前任务的其他任务的入度
			for _, otherTask := range remaining {
				for _, depID := range otherTask.Dependencies {
					if depID == task.ID {
						inDegree[otherTask.ID]--
					}
				}
			}
		}
	}

	return levels, nil
}

// Validate 验证依赖关系
func (r *DefaultDependencyResolver) Validate(tasks []*Task) error {
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks to validate")
	}

	// 构建任务 ID 集合
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		if task.ID == "" {
			return fmt.Errorf("task ID cannot be empty")
		}
		if taskIDs[task.ID] {
			return fmt.Errorf("duplicate task ID: %s", task.ID)
		}
		taskIDs[task.ID] = true
	}

	// 验证依赖关系
	for _, task := range tasks {
		for _, depID := range task.Dependencies {
			if depID == task.ID {
				return fmt.Errorf("task %s cannot depend on itself", task.ID)
			}
			if !taskIDs[depID] {
				return fmt.Errorf("task %s depends on non-existent task %s", task.ID, depID)
			}
		}
	}

	// 检查循环依赖
	if err := r.checkCyclicDependency(tasks); err != nil {
		return err
	}

	return nil
}

// checkCyclicDependency 检查循环依赖
func (r *DefaultDependencyResolver) checkCyclicDependency(tasks []*Task) error {
	// 使用 DFS 检测循环依赖
	taskMap := make(map[string]*Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(taskID string) bool
	dfs = func(taskID string) bool {
		visited[taskID] = true
		recStack[taskID] = true

		task := taskMap[taskID]
		for _, depID := range task.Dependencies {
			if !visited[depID] {
				if dfs(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[taskID] = false
		return false
	}

	for _, task := range tasks {
		if !visited[task.ID] {
			if dfs(task.ID) {
				return fmt.Errorf("circular dependency detected starting from task: %s", task.ID)
			}
		}
	}

	return nil
}

// getTaskIDs 获取任务 ID 列表
func getTaskIDs(taskMap map[string]*Task) []string {
	var ids []string
	for id := range taskMap {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// GetExecutionOrder 获取任务执行顺序（扁平化）
func (r *DefaultDependencyResolver) GetExecutionOrder(tasks []*Task) ([]*Task, error) {
	levels, err := r.Resolve(tasks)
	if err != nil {
		return nil, err
	}

	var order []*Task
	for _, level := range levels {
		order = append(order, level...)
	}

	return order, nil
}

// GetParallelGroups 获取可并行执行的任务组
func (r *DefaultDependencyResolver) GetParallelGroups(tasks []*Task) ([][]*Task, error) {
	return r.Resolve(tasks)
}

// CanExecuteInParallel 检查两个任务是否可以并行执行
func (r *DefaultDependencyResolver) CanExecuteInParallel(task1, task2 *Task, allTasks []*Task) (bool, error) {
	// 构建任务映射
	taskMap := make(map[string]*Task)
	for _, task := range allTasks {
		taskMap[task.ID] = task
	}

	// 检查直接依赖
	if r.hasDependency(task1, task2.ID, taskMap) || r.hasDependency(task2, task1.ID, taskMap) {
		return false, nil
	}

	// 检查间接依赖
	if r.hasTransitiveDependency(task1, task2.ID, taskMap, make(map[string]bool)) ||
		r.hasTransitiveDependency(task2, task1.ID, taskMap, make(map[string]bool)) {
		return false, nil
	}

	return true, nil
}

// hasDependency 检查任务是否直接依赖另一个任务
func (r *DefaultDependencyResolver) hasDependency(task *Task, targetID string, taskMap map[string]*Task) bool {
	for _, depID := range task.Dependencies {
		if depID == targetID {
			return true
		}
	}
	return false
}

// hasTransitiveDependency 检查任务是否传递依赖另一个任务
func (r *DefaultDependencyResolver) hasTransitiveDependency(task *Task, targetID string, taskMap map[string]*Task, visited map[string]bool) bool {
	if visited[task.ID] {
		return false
	}
	visited[task.ID] = true

	for _, depID := range task.Dependencies {
		if depID == targetID {
			return true
		}
		if depTask, exists := taskMap[depID]; exists {
			if r.hasTransitiveDependency(depTask, targetID, taskMap, visited) {
				return true
			}
		}
	}

	return false
}

// GetDependencyGraph 获取依赖图的邻接表表示
func (r *DefaultDependencyResolver) GetDependencyGraph(tasks []*Task) map[string][]string {
	graph := make(map[string][]string)

	for _, task := range tasks {
		graph[task.ID] = make([]string, len(task.Dependencies))
		copy(graph[task.ID], task.Dependencies)
	}

	return graph
}

// GetReverseDependencyGraph 获取反向依赖图（谁依赖我）
func (r *DefaultDependencyResolver) GetReverseDependencyGraph(tasks []*Task) map[string][]string {
	reverseGraph := make(map[string][]string)

	// 初始化
	for _, task := range tasks {
		reverseGraph[task.ID] = make([]string, 0)
	}

	// 构建反向依赖
	for _, task := range tasks {
		for _, depID := range task.Dependencies {
			reverseGraph[depID] = append(reverseGraph[depID], task.ID)
		}
	}

	return reverseGraph
}

// GetCriticalPath 获取关键路径（最长依赖链）
func (r *DefaultDependencyResolver) GetCriticalPath(tasks []*Task) ([]*Task, error) {
	taskMap := make(map[string]*Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	// 计算每个任务的最长路径
	memo := make(map[string]int)
	pathMemo := make(map[string][]*Task)

	var dfs func(taskID string) (int, []*Task)
	dfs = func(taskID string) (int, []*Task) {
		if length, exists := memo[taskID]; exists {
			return length, pathMemo[taskID]
		}

		task := taskMap[taskID]
		maxLength := 0
		var longestPath []*Task

		for _, depID := range task.Dependencies {
			length, path := dfs(depID)
			if length > maxLength {
				maxLength = length
				longestPath = path
			}
		}

		currentLength := maxLength + 1
		currentPath := append([]*Task{task}, longestPath...)

		memo[taskID] = currentLength
		pathMemo[taskID] = currentPath

		return currentLength, currentPath
	}

	// 找到最长路径
	maxLength := 0
	var criticalPath []*Task

	for _, task := range tasks {
		length, path := dfs(task.ID)
		if length > maxLength {
			maxLength = length
			criticalPath = path
		}
	}

	return criticalPath, nil
}
