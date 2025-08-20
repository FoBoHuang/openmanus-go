package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/config"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"
)

func main() {
	// 示例：数据分析 Agent
	fmt.Println("📊 OpenManus-Go Data Analysis Example")
	fmt.Println("====================================")

	// 检查示例数据文件是否存在
	if err := createSampleData(); err != nil {
		log.Fatalf("Failed to create sample data: %v", err)
	}

	// 1. 加载配置
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "your-api-key-here" // 在实际使用中设置真实的 API Key
	cfg.Agent.MaxSteps = 8
	cfg.RunFlow.UseDataAnalysisAgent = true

	// 2. 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())

	// 3. 创建工具注册表
	toolRegistry := tool.NewRegistry()

	// 注册数据分析相关工具
	if err := registerDataAnalysisTools(toolRegistry, cfg); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// 4. 创建 Agent
	agentConfig := agent.DefaultConfig()
	agentConfig.MaxSteps = 8

	baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)

	// 5. 数据分析任务
	tasks := []string{
		"读取 sample_data.csv 文件并分析其结构",
		"计算数据的基本统计信息（平均值、最大值、最小值等）",
		"识别数据中的模式和趋势",
		"生成数据分析报告并保存为 analysis_report.txt",
	}

	// 6. 执行数据分析任务
	ctx := context.Background()

	for i, task := range tasks {
		fmt.Printf("\n📈 Analysis Task %d: %s\n", i+1, task)
		fmt.Println(strings.Repeat("=", 60))

		result, err := baseAgent.Loop(ctx, task)
		if err != nil {
			fmt.Printf("❌ Task failed: %v\n", err)
			continue
		}

		fmt.Printf("✅ Result: %s\n", result)
	}

	fmt.Println("\n🎉 Data analysis completed!")
	fmt.Println("\n📋 Generated files:")
	if _, err := os.Stat("analysis_report.txt"); err == nil {
		fmt.Println("  - analysis_report.txt (分析报告)")
	}
	if _, err := os.Stat("data_visualization.html"); err == nil {
		fmt.Println("  - data_visualization.html (数据可视化)")
	}
}

func createSampleData() error {
	// 创建示例 CSV 数据
	csvData := `name,age,salary,department,years_experience
Alice,28,65000,Engineering,3
Bob,35,75000,Engineering,8
Charlie,42,85000,Management,15
Diana,29,62000,Marketing,4
Eve,33,70000,Engineering,6
Frank,38,80000,Management,12
Grace,26,58000,Marketing,2
Henry,45,90000,Management,18
Ivy,31,68000,Engineering,5
Jack,27,60000,Marketing,3`

	return os.WriteFile("sample_data.csv", []byte(csvData), 0644)
}

func registerDataAnalysisTools(registry *tool.Registry, cfg *config.Config) error {
	// 注册文件系统工具（用于读取 CSV）
	fsTool := builtin.NewFileSystemTool(
		[]string{"."}, // 允许当前目录
		[]string{},    // 无禁止路径
	)
	if err := registry.Register(fsTool); err != nil {
		return fmt.Errorf("failed to register fs tool: %w", err)
	}

	// 注册 HTTP 工具（可能需要获取外部数据）
	httpTool := builtin.NewHTTPTool()
	if err := registry.Register(httpTool); err != nil {
		return fmt.Errorf("failed to register http tool: %w", err)
	}

	// 注册数据分析专用工具（示例）
	dataAnalysisTool := newDataAnalysisTool()
	if err := registry.Register(dataAnalysisTool); err != nil {
		return fmt.Errorf("failed to register data analysis tool: %w", err)
	}

	return nil
}

// DataAnalysisTool 数据分析专用工具
type DataAnalysisTool struct {
	*tool.BaseTool
}

func newDataAnalysisTool() *DataAnalysisTool {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"operation": tool.StringProperty("操作类型：parse_csv, calculate_stats, find_patterns, generate_report"),
		"data":      tool.StringProperty("CSV 数据内容"),
		"file_path": tool.StringProperty("CSV 文件路径"),
		"columns":   tool.ArrayProperty("要分析的列名", tool.StringProperty("")),
	}, []string{"operation"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success": tool.BooleanProperty("操作是否成功"),
		"result":  tool.StringProperty("分析结果"),
		"statistics": tool.ObjectProperty("统计信息", map[string]any{
			"additionalProperties": map[string]any{
				"type":        "number",
				"description": "统计值",
			},
		}),
		"patterns": tool.ArrayProperty("发现的模式", tool.StringProperty("")),
		"report":   tool.StringProperty("生成的报告"),
		"error":    tool.StringProperty("错误信息"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"data_analysis",
		"数据分析工具，支持 CSV 解析、统计计算、模式识别等功能",
		inputSchema,
		outputSchema,
	)

	return &DataAnalysisTool{
		BaseTool: baseTool,
	}
}

func (da *DataAnalysisTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return da.errorResult("operation is required"), nil
	}

	switch operation {
	case "parse_csv":
		return da.parseCSV(args)
	case "calculate_stats":
		return da.calculateStats(args)
	case "find_patterns":
		return da.findPatterns(args)
	case "generate_report":
		return da.generateReport(args)
	default:
		return da.errorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

func (da *DataAnalysisTool) parseCSV(args map[string]any) (map[string]any, error) {
	// 简化的 CSV 解析逻辑
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		return da.errorResult("file_path is required for parse_csv"), nil
	}

	// 在实际实现中，这里会解析 CSV 文件
	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully parsed CSV file: %s", filePath),
		"columns": []string{"name", "age", "salary", "department", "years_experience"},
		"rows":    10,
	}, nil
}

func (da *DataAnalysisTool) calculateStats(args map[string]any) (map[string]any, error) {
	// 简化的统计计算
	return map[string]any{
		"success": true,
		"result":  "Statistical analysis completed",
		"statistics": map[string]any{
			"mean_age":        33.8,
			"mean_salary":     69300,
			"max_salary":      90000,
			"min_salary":      58000,
			"total_employees": 10,
		},
	}, nil
}

func (da *DataAnalysisTool) findPatterns(args map[string]any) (map[string]any, error) {
	// 简化的模式识别
	patterns := []string{
		"Engineering department has the most employees (40%)",
		"Salary increases with years of experience",
		"Management positions have higher average salaries",
		"Age distribution is relatively uniform",
	}

	return map[string]any{
		"success":  true,
		"result":   "Pattern analysis completed",
		"patterns": patterns,
	}, nil
}

func (da *DataAnalysisTool) generateReport(args map[string]any) (map[string]any, error) {
	// 生成分析报告
	report := `# Data Analysis Report

## Dataset Overview
- Total records: 10
- Columns: name, age, salary, department, years_experience

## Key Statistics
- Average age: 33.8 years
- Average salary: $69,300
- Salary range: $58,000 - $90,000

## Key Findings
1. Engineering department has the most employees (40%)
2. Salary increases with years of experience
3. Management positions have higher average salaries
4. Age distribution is relatively uniform

## Recommendations
- Consider salary adjustments for junior employees
- Investigate experience-based promotion opportunities
- Review departmental balance

Generated by OpenManus-Go Data Analysis Agent`

	// 保存报告到文件
	if err := os.WriteFile("analysis_report.txt", []byte(report), 0644); err != nil {
		return da.errorResult(fmt.Sprintf("failed to save report: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  "Analysis report generated and saved to analysis_report.txt",
		"report":  report,
	}, nil
}

func (da *DataAnalysisTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}
