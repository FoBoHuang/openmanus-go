package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"openmanus-go/pkg/agent"
	"openmanus-go/pkg/config"
	"openmanus-go/pkg/llm"
	"openmanus-go/pkg/tool"
	"openmanus-go/pkg/tool/builtin"

	"github.com/spf13/cobra"
)

// NewRunCommand 创建运行命令
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [goal]",
		Short: "运行单个 Agent 执行指定目标",
		Long: `运行单个 Agent 来执行用户指定的目标。

Agent 将通过 Plan -> Tool Use -> Observation -> Reflection -> Next Action 的循环来完成任务。

示例:
  openmanus run "搜索最新的 Go 语言新闻"
  openmanus run "分析 data.csv 文件并生成报告"
  openmanus run --interactive`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAgent,
	}

	// 添加命令特定的标志
	cmd.Flags().BoolP("interactive", "i", false, "交互模式")
	cmd.Flags().StringP("output", "o", "", "输出文件路径")
	cmd.Flags().IntP("max-steps", "s", 0, "最大步数（0 表示使用配置默认值）")
	cmd.Flags().IntP("max-tokens", "t", 0, "最大 token 数（0 表示使用配置默认值）")
	cmd.Flags().StringP("temperature", "T", "", "LLM 温度（0.0-2.0）")
	cmd.Flags().BoolP("save-trace", "S", true, "保存执行轨迹")

	return cmd
}

func runAgent(cmd *cobra.Command, args []string) error {
	// 获取配置路径
	configPath, _ := cmd.Flags().GetString("config")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 设置日志级别
	if debug {
		fmt.Println("Debug mode enabled")
	} else if verbose {
		fmt.Println("Verbose mode enabled")
	}

	// 获取目标
	var goal string
	interactive, _ := cmd.Flags().GetBool("interactive")

	if len(args) > 0 {
		goal = args[0]
	} else if !interactive {
		return fmt.Errorf("goal is required in non-interactive mode")
	}

	// 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(cfg.ToLLMConfig())

	// 应用命令行覆盖
	if temp, _ := cmd.Flags().GetString("temperature"); temp != "" {
		// 解析并设置温度
		fmt.Printf("Setting temperature to %s\n", temp)
	}

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()
	if err := builtin.RegisterBuiltinTools(toolRegistry, cfg); err != nil {
		return fmt.Errorf("failed to register builtin tools: %w", err)
	}

	// 创建 Agent 配置
	agentConfig := agent.DefaultConfig()
	if maxSteps, _ := cmd.Flags().GetInt("max-steps"); maxSteps > 0 {
		agentConfig.MaxSteps = maxSteps
	}
	if maxTokens, _ := cmd.Flags().GetInt("max-tokens"); maxTokens > 0 {
		agentConfig.MaxTokens = maxTokens
	}

	// 创建 Agent
	baseAgent := agent.NewBaseAgent(llmClient, toolRegistry, agentConfig)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, stopping...")
		cancel()
	}()

	if interactive {
		return runInteractiveMode(ctx, baseAgent, cmd)
	} else {
		return runSingleGoal(ctx, baseAgent, goal, cmd)
	}
}

func runSingleGoal(ctx context.Context, agent agent.Agent, goal string, cmd *cobra.Command) error {
	fmt.Printf("🎯 Goal: %s\n\n", goal)

	// 执行任务
	result, err := agent.Loop(ctx, goal)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// 输出结果
	fmt.Printf("✅ Result:\n%s\n", result)

	// 保存轨迹
	saveTrace, _ := cmd.Flags().GetBool("save-trace")
	if saveTrace {
		// TODO: 实现轨迹保存功能
		fmt.Println("📝 Trace saving not implemented yet")
	}

	// 保存输出到文件
	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("💾 Output saved to %s\n", outputPath)
	}

	return nil
}

func runInteractiveMode(ctx context.Context, agent agent.Agent, cmd *cobra.Command) error {
	fmt.Println("🤖 OpenManus-Go Interactive Mode")
	fmt.Println("Type your goals and press Enter. Type 'quit' or 'exit' to stop.")
	fmt.Println("Commands: /help, /status, /trace, /config")
	fmt.Println()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Goodbye!")
			return nil
		default:
		}

		// 读取用户输入
		fmt.Print("🎯 Goal: ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			continue
		}

		// 处理特殊命令
		switch input {
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return nil
		case "/help":
			printHelp()
			continue
		case "/status":
			printStatus(agent)
			continue
		case "/trace":
			printTrace(agent)
			continue
		case "/config":
			printConfig()
			continue
		case "":
			continue
		}

		// 执行目标
		fmt.Printf("\n🔄 Executing: %s\n", input)
		result, err := agent.Loop(ctx, input)
		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}

		fmt.Printf("✅ Result:\n%s\n\n", result)

		// 自动保存轨迹
		// TODO: 实现轨迹自动保存
	}
}

func printHelp() {
	fmt.Println(`
Available commands:
  /help    - Show this help message
  /status  - Show agent status
  /trace   - Show current execution trace
  /config  - Show configuration
  quit     - Exit the program
  exit     - Exit the program
`)
}

func printStatus(agent agent.Agent) {
	fmt.Printf(`
Agent Status:
  Status: Running
  Type: BaseAgent
`)
}

func printTrace(agent agent.Agent) {
	fmt.Println(`
Current Trace:
  No trace information available yet
`)
}

func printConfig() {
	fmt.Println(`
Configuration:
  Config file: Use --config to specify
  Tools: HTTP, FileSystem, Redis, MySQL, Browser, Crawler
  Storage: File-based trace storage
  
Use 'openmanus config show' for detailed configuration.
`)
}
