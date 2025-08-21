package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/flow"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"

	"github.com/spf13/cobra"
)

// NewFlowCommand 创建流程命令
func NewFlowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flow",
		Short: "多 Agent 流程编排",
		Long: `运行多 Agent 协作流程，支持复杂的任务编排和工作流。

多 Agent 流程允许：
- 任务分解和并行处理
- Agent 之间的协作和信息共享
- 复杂工作流的编排和执行
- 数据分析 Agent 的集成

示例:
  openmanus flow --workflow data-analysis
  openmanus flow --config workflow.yaml`,
		RunE: runFlow,
	}

	cmd.Flags().StringP("workflow", "w", "", "工作流名称或配置文件")
	cmd.Flags().Bool("data-analysis", false, "启用数据分析 Agent")
	cmd.Flags().IntP("agents", "a", 2, "Agent 数量")
	cmd.Flags().StringP("mode", "m", "sequential", "执行模式 (sequential, parallel, dag)")

	return cmd
}

func runFlow(cmd *cobra.Command, args []string) error {
	workflowName, _ := cmd.Flags().GetString("workflow")
	dataAnalysis, _ := cmd.Flags().GetBool("data-analysis")
	agentCount, _ := cmd.Flags().GetInt("agents")
	mode, _ := cmd.Flags().GetString("mode")

	logger.Info("🔄 Starting Multi-Agent Flow")
	logger.Infof("   Workflow: %s", getWorkflowName(workflowName))
	logger.Infof("   Mode: %s", mode)
	logger.Infof("   Agents: %d", agentCount)
	logger.Infof("   Data Analysis: %t", dataAnalysis)
	logger.Info("")

	// 加载配置
	cfg := config.DefaultConfig()

	// 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()
	if err := builtin.RegisterBuiltinTools(toolRegistry, cfg); err != nil {
		return fmt.Errorf("failed to register builtin tools: %w", err)
	}

	// 创建 Agent 工厂
	agentFactory := flow.NewDefaultAgentFactory(llmClient, toolRegistry)

	// 创建流程引擎
	flowEngine := flow.NewDefaultFlowEngine(agentFactory, 5) // 最大并发数为 5

	// 根据参数创建示例工作流
	var workflow *flow.Workflow
	if workflowName != "" {
		var err error
		workflow, err = loadWorkflowFromFile(workflowName)
		if err != nil {
			logger.Warnf("⚠️  Failed to load workflow from file, creating demo workflow: %v", err)
			workflow = createDemoWorkflow(mode, dataAnalysis, agentCount)
		}
	} else {
		workflow = createDemoWorkflow(mode, dataAnalysis, agentCount)
	}

	logger.Infof("📋 Workflow: %s (%d tasks)", workflow.Name, len(workflow.Tasks))
	logger.Infof("🔧 Execution Mode: %s", workflow.Mode)
	logger.Info("")

	// 执行工作流
	ctx := context.Background()
	input := map[string]interface{}{
		"demo_mode": true,
		"timestamp": time.Now(),
	}

	execution, err := flowEngine.Execute(ctx, workflow, input)
	if err != nil {
		return fmt.Errorf("failed to start workflow execution: %w", err)
	}

	logger.Infof("🚀 Workflow execution started (ID: %s)", execution.ID)

	// 监听执行事件
	eventChan, err := flowEngine.Subscribe(execution.ID)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// 等待执行完成
	timeout := time.After(5 * time.Minute) // 5分钟超时
	for {
		select {
		case event := <-eventChan:
			if event == nil {
				// 通道已关闭，执行完成
				goto done
			}
			printFlowEvent(event)

		case <-timeout:
			logger.Info("⏰ Execution timeout, canceling...")
			flowEngine.CancelExecution(execution.ID)
			return fmt.Errorf("workflow execution timeout")

		case <-time.After(100 * time.Millisecond):
			// 检查执行状态
			currentExecution, err := flowEngine.GetExecution(execution.ID)
			if err != nil {
				return fmt.Errorf("failed to get execution status: %w", err)
			}

			if currentExecution.Status != flow.FlowStatusRunning {
				goto done
			}
		}
	}

done:
	// 获取最终结果
	finalExecution, err := flowEngine.GetExecution(execution.ID)
	if err != nil {
		return fmt.Errorf("failed to get final execution status: %w", err)
	}

	// 打印结果
	printFlowResult(finalExecution)

	// 清理资源
	if err := flowEngine.Cleanup(execution.ID); err != nil {
		logger.Warnf("⚠️  Warning: Failed to cleanup execution: %v", err)
	}

	return nil
}

func getWorkflowName(workflow string) string {
	if workflow == "" {
		return "default"
	}
	return workflow
}

// createDemoWorkflow 创建演示工作流
func createDemoWorkflow(mode string, dataAnalysis bool, agentCount int) *flow.Workflow {
	var executionMode flow.ExecutionMode
	switch mode {
	case "parallel":
		executionMode = flow.ExecutionModeParallel
	case "dag":
		executionMode = flow.ExecutionModeDAG
	default:
		executionMode = flow.ExecutionModeSequential
	}

	workflow := flow.NewWorkflow("demo-workflow", "Demo Multi-Agent Workflow", executionMode)

	if dataAnalysis {
		// 数据分析工作流
		task1 := flow.NewTask("fetch-data", "获取数据", "general", "从网络获取一些示例数据")
		task1.Input["url"] = "https://jsonplaceholder.typicode.com/posts/1"

		task2 := flow.NewTask("analyze-data", "分析数据", "data_analysis", "分析获取到的数据并生成报告")
		task2.Dependencies = []string{"fetch-data"}

		task3 := flow.NewTask("save-report", "保存报告", "file_processor", "将分析报告保存到文件")
		task3.Dependencies = []string{"analyze-data"}

		workflow.AddTask(task1)
		workflow.AddTask(task2)
		workflow.AddTask(task3)
	} else {
		// 通用工作流
		for i := 0; i < agentCount; i++ {
			taskID := fmt.Sprintf("task-%d", i+1)
			taskName := fmt.Sprintf("任务 %d", i+1)
			agentType := "general"

			if i%2 == 1 {
				agentType = "web_scraper"
			}

			goal := fmt.Sprintf("执行第 %d 个任务：创建一个简单的文本文件", i+1)
			task := flow.NewTask(taskID, taskName, agentType, goal)

			// 添加一些依赖关系（DAG 模式）
			if mode == "dag" && i > 0 {
				if i == 1 {
					task.Dependencies = []string{"task-1"}
				} else if i == 2 {
					task.Dependencies = []string{"task-1"}
				} else {
					task.Dependencies = []string{fmt.Sprintf("task-%d", i)}
				}
			}

			workflow.AddTask(task)
		}
	}

	return workflow
}

// loadWorkflowFromFile 从文件加载工作流（占位符实现）
func loadWorkflowFromFile(filename string) (*flow.Workflow, error) {
	// TODO: 实现从 YAML/JSON 文件加载工作流
	return nil, fmt.Errorf("workflow file loading not implemented yet")
}

// printFlowEvent 打印流程事件
func printFlowEvent(event *flow.FlowEvent) {
	timestamp := event.Timestamp.Format("15:04:05")

	switch event.Type {
	case flow.FlowEventTypeFlowStarted:
		logger.Infof("[%s] 🚀 Flow started: %s", timestamp, event.Message)
	case flow.FlowEventTypeFlowCompleted:
		logger.Infof("[%s] ✅ Flow completed: %s", timestamp, event.Message)
	case flow.FlowEventTypeFlowFailed:
		logger.Infof("[%s] ❌ Flow failed: %s", timestamp, event.Message)
	case flow.FlowEventTypeFlowCanceled:
		logger.Infof("[%s] 🛑 Flow canceled: %s", timestamp, event.Message)
	case flow.FlowEventTypeTaskStarted:
		logger.Infof("[%s] 🔄 Task started: %s (ID: %s)", timestamp, event.Message, event.TaskID)
	case flow.FlowEventTypeTaskCompleted:
		logger.Infof("[%s] ✅ Task completed: %s (ID: %s)", timestamp, event.Message, event.TaskID)
	case flow.FlowEventTypeTaskFailed:
		logger.Infof("[%s] ❌ Task failed: %s (ID: %s)", timestamp, event.Message, event.TaskID)
	case flow.FlowEventTypeTaskSkipped:
		logger.Infof("[%s] ⏭️  Task skipped: %s (ID: %s)", timestamp, event.Message, event.TaskID)
	default:
		logger.Infof("[%s] 📝 Event: %s", timestamp, event.Message)
	}
}

// printFlowResult 打印流程结果
func printFlowResult(execution *flow.FlowExecution) {
	logger.Info("\n" + strings.Repeat("=", 60))
	logger.Info("📊 Workflow Execution Summary")
	logger.Info(strings.Repeat("=", 60))

	logger.Infof("Flow ID: %s", execution.ID)
	logger.Infof("Workflow: %s", execution.Workflow.Name)
	logger.Infof("Status: %s", execution.Status)
	logger.Infof("Mode: %s", execution.Workflow.Mode)

	if execution.StartTime != nil {
		logger.Infof("Started: %s", execution.StartTime.Format("2006-01-02 15:04:05"))
	}

	if execution.EndTime != nil {
		logger.Infof("Completed: %s", execution.EndTime.Format("2006-01-02 15:04:05"))
		logger.Infof("Duration: %.2f seconds", execution.Duration.Seconds())
	}

	if execution.Error != "" {
		logger.Infof("Error: %s", execution.Error)
	}

	logger.Info("\n📋 Task Results:")
	for i, task := range execution.Workflow.Tasks {
		status := "❓"
		switch task.Status {
		case flow.TaskStatusCompleted:
			status = "✅"
		case flow.TaskStatusFailed:
			status = "❌"
		case flow.TaskStatusSkipped:
			status = "⏭️"
		case flow.TaskStatusRunning:
			status = "🔄"
		case flow.TaskStatusPending:
			status = "⏳"
		case flow.TaskStatusCanceled:
			status = "🛑"
		}

		logger.Infof("  %d. %s %s (%s)", i+1, status, task.Name, task.ID)

		if task.Error != "" {
			logger.Infof("     Error: %s", task.Error)
		}

		if task.Duration > 0 {
			logger.Infof("     Duration: %.2f seconds", task.Duration.Seconds())
		}

		if task.Trace != nil && len(task.Trace.Steps) > 0 {
			logger.Infof("     Steps: %d", len(task.Trace.Steps))
		}
	}

	// 打印统计信息
	if stats, ok := execution.Output["stats"].(map[string]interface{}); ok {
		logger.Info("\n📈 Statistics:")
		if totalTasks, ok := stats["total_tasks"].(int); ok {
			logger.Infof("  Total Tasks: %d", totalTasks)
		}
		if completedTasks, ok := stats["completed_tasks"].(int); ok {
			logger.Infof("  Completed: %d", completedTasks)
		}
		if failedTasks, ok := stats["failed_tasks"].(int); ok {
			logger.Infof("  Failed: %d", failedTasks)
		}
		if successRate, ok := stats["success_rate"].(float64); ok {
			logger.Infof("  Success Rate: %.1f%%", successRate*100)
		}
		if totalSteps, ok := stats["total_steps"].(int); ok {
			logger.Infof("  Total Steps: %d", totalSteps)
		}
	}

	logger.Info(strings.Repeat("=", 60))
}
