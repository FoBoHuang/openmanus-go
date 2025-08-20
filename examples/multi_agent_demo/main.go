package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/flow"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

func main() {
	fmt.Println("🤖 OpenManus-Go Multi-Agent Demo")
	fmt.Println("=================================")

	// 加载配置
	cfg := config.DefaultConfig()

	// 注意：在实际使用中，请设置真实的 API 密钥
	// cfg.LLM.APIKey = "your-openai-api-key"

	// 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()
	if err := builtin.RegisterBuiltinTools(toolRegistry, cfg); err != nil {
		log.Fatalf("Failed to register builtin tools: %v", err)
	}

	// 创建 Agent 工厂
	agentFactory := flow.NewDefaultAgentFactory(llmClient, toolRegistry)

	// 创建流程引擎
	flowEngine := flow.NewDefaultFlowEngine(agentFactory, 3)

	// 创建示例工作流
	workflow := createDemoWorkflow()

	fmt.Printf("📋 Workflow: %s\n", workflow.Name)
	fmt.Printf("🔧 Mode: %s\n", workflow.Mode)
	fmt.Printf("📝 Tasks: %d\n", len(workflow.Tasks))
	fmt.Println()

	// 打印任务信息
	fmt.Println("📋 Task Details:")
	for i, task := range workflow.Tasks {
		fmt.Printf("  %d. %s (%s) - Agent: %s\n", i+1, task.Name, task.ID, task.AgentType)
		if len(task.Dependencies) > 0 {
			fmt.Printf("     Dependencies: %v\n", task.Dependencies)
		}
	}
	fmt.Println()

	// 执行工作流
	ctx := context.Background()
	input := map[string]interface{}{
		"demo_mode": true,
		"timestamp": time.Now(),
		"user":      "demo-user",
	}

	fmt.Println("🚀 Starting workflow execution...")
	execution, err := flowEngine.Execute(ctx, workflow, input)
	if err != nil {
		log.Fatalf("Failed to start workflow: %v", err)
	}

	fmt.Printf("🆔 Execution ID: %s\n", execution.ID)

	// 监听执行事件
	eventChan, err := flowEngine.Subscribe(execution.ID)
	if err != nil {
		log.Fatalf("Failed to subscribe to events: %v", err)
	}

	// 等待执行完成（最多 2 分钟）
	timeout := time.After(2 * time.Minute)
	eventCount := 0

	for {
		select {
		case event := <-eventChan:
			if event == nil {
				// 通道已关闭，执行完成
				goto done
			}
			eventCount++
			printEvent(event)

		case <-timeout:
			fmt.Println("⏰ Execution timeout, canceling...")
			flowEngine.CancelExecution(execution.ID)
			goto done

		case <-time.After(100 * time.Millisecond):
			// 检查执行状态
			currentExecution, err := flowEngine.GetExecution(execution.ID)
			if err != nil {
				log.Printf("Failed to get execution status: %v", err)
				goto done
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
		log.Fatalf("Failed to get final execution: %v", err)
	}

	// 打印执行摘要
	printExecutionSummary(finalExecution, eventCount)

	// 清理资源
	if err := flowEngine.Cleanup(execution.ID); err != nil {
		log.Printf("Warning: Failed to cleanup: %v", err)
	}

	fmt.Println("\n🎉 Multi-Agent Demo completed!")
}

// createDemoWorkflow 创建演示工作流
func createDemoWorkflow() *flow.Workflow {
	workflow := flow.NewWorkflow("demo-workflow", "Multi-Agent Demo Workflow", flow.ExecutionModeDAG)

	// 任务 1: 数据收集
	task1 := flow.NewTask("collect-data", "数据收集", "general", "收集一些示例数据用于后续处理")
	task1.Input["source"] = "demo"
	task1.Input["format"] = "json"

	// 任务 2: 数据处理
	task2 := flow.NewTask("process-data", "数据处理", "data_analysis", "处理收集到的数据并进行分析")
	task2.Dependencies = []string{"collect-data"}
	task2.Input["analysis_type"] = "basic"

	// 任务 3: 网页内容获取（并行任务）
	task3 := flow.NewTask("fetch-web-content", "网页内容获取", "web_scraper", "获取网页内容用于分析")
	task3.Input["url"] = "https://httpbin.org/json"

	// 任务 4: 报告生成
	task4 := flow.NewTask("generate-report", "生成报告", "file_processor", "基于处理结果生成最终报告")
	task4.Dependencies = []string{"process-data", "fetch-web-content"}

	// 任务 5: 结果汇总
	task5 := flow.NewTask("summarize-results", "结果汇总", "general", "汇总所有任务的结果")
	task5.Dependencies = []string{"generate-report"}

	workflow.AddTask(task1)
	workflow.AddTask(task2)
	workflow.AddTask(task3)
	workflow.AddTask(task4)
	workflow.AddTask(task5)

	return workflow
}

// printEvent 打印事件
func printEvent(event *flow.FlowEvent) {
	timestamp := event.Timestamp.Format("15:04:05")

	switch event.Type {
	case flow.FlowEventTypeFlowStarted:
		fmt.Printf("[%s] 🚀 %s\n", timestamp, event.Message)
	case flow.FlowEventTypeFlowCompleted:
		fmt.Printf("[%s] ✅ %s\n", timestamp, event.Message)
	case flow.FlowEventTypeFlowFailed:
		fmt.Printf("[%s] ❌ %s\n", timestamp, event.Message)
	case flow.FlowEventTypeFlowCanceled:
		fmt.Printf("[%s] 🛑 %s\n", timestamp, event.Message)
	case flow.FlowEventTypeTaskStarted:
		fmt.Printf("[%s] 🔄 %s\n", timestamp, event.Message)
	case flow.FlowEventTypeTaskCompleted:
		fmt.Printf("[%s] ✅ %s\n", timestamp, event.Message)
	case flow.FlowEventTypeTaskFailed:
		fmt.Printf("[%s] ❌ %s\n", timestamp, event.Message)
	case flow.FlowEventTypeTaskSkipped:
		fmt.Printf("[%s] ⏭️  %s\n", timestamp, event.Message)
	default:
		fmt.Printf("[%s] 📝 %s\n", timestamp, event.Message)
	}
}

// printExecutionSummary 打印执行摘要
func printExecutionSummary(execution *flow.FlowExecution, eventCount int) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 Execution Summary")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("🆔 Flow ID: %s\n", execution.ID)
	fmt.Printf("📋 Workflow: %s\n", execution.Workflow.Name)
	fmt.Printf("📊 Status: %s\n", execution.Status)
	fmt.Printf("🔧 Mode: %s\n", execution.Workflow.Mode)
	fmt.Printf("📨 Events: %d\n", eventCount)

	if execution.StartTime != nil {
		fmt.Printf("⏰ Started: %s\n", execution.StartTime.Format("15:04:05"))
	}

	if execution.EndTime != nil {
		fmt.Printf("🏁 Completed: %s\n", execution.EndTime.Format("15:04:05"))
		fmt.Printf("⏱️  Duration: %.2f seconds\n", execution.Duration.Seconds())
	}

	if execution.Error != "" {
		fmt.Printf("❌ Error: %s\n", execution.Error)
	}

	fmt.Println("\n📋 Task Results:")
	completed := 0
	failed := 0

	for i, task := range execution.Workflow.Tasks {
		status := "❓"
		switch task.Status {
		case flow.TaskStatusCompleted:
			status = "✅"
			completed++
		case flow.TaskStatusFailed:
			status = "❌"
			failed++
		case flow.TaskStatusSkipped:
			status = "⏭️"
		case flow.TaskStatusRunning:
			status = "🔄"
		case flow.TaskStatusPending:
			status = "⏳"
		case flow.TaskStatusCanceled:
			status = "🛑"
		}

		fmt.Printf("  %d. %s %s (%s)\n", i+1, status, task.Name, task.AgentType)

		if task.Error != "" {
			fmt.Printf("     ❌ %s\n", task.Error)
		}

		if task.Duration > 0 {
			fmt.Printf("     ⏱️  %.2fs\n", task.Duration.Seconds())
		}
	}

	fmt.Printf("\n📈 Statistics:\n")
	fmt.Printf("  ✅ Completed: %d/%d\n", completed, len(execution.Workflow.Tasks))
	fmt.Printf("  ❌ Failed: %d/%d\n", failed, len(execution.Workflow.Tasks))
	fmt.Printf("  📊 Success Rate: %.1f%%\n", float64(completed)/float64(len(execution.Workflow.Tasks))*100)

	fmt.Println(strings.Repeat("=", 60))
}
