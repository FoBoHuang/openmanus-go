package commands

import (
	"fmt"
	"strings"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/state"

	"github.com/spf13/cobra"
)

// NewTraceCommand 创建轨迹管理命令
func NewTraceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "管理执行轨迹",
		Long: `管理和查看保存的执行轨迹。

子命令:
  list     - 列出所有保存的轨迹
  show     - 显示特定轨迹的详细信息
  delete   - 删除指定的轨迹
  clean    - 清理旧的轨迹文件`,
	}

	// 添加子命令
	cmd.AddCommand(newTraceListCommand())
	cmd.AddCommand(newTraceShowCommand())
	cmd.AddCommand(newTraceDeleteCommand())
	cmd.AddCommand(newTraceCleanCommand())

	return cmd
}

// newTraceListCommand 创建列出轨迹的命令
func newTraceListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有保存的轨迹",
		RunE:  runTraceList,
	}

	cmd.Flags().IntP("limit", "l", 10, "显示的轨迹数量限制")
	cmd.Flags().BoolP("verbose", "v", false, "显示详细信息")

	return cmd
}

// newTraceShowCommand 创建显示轨迹的命令
func newTraceShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <trace-id>",
		Short: "显示特定轨迹的详细信息",
		Args:  cobra.ExactArgs(1),
		RunE:  runTraceShow,
	}

	cmd.Flags().BoolP("steps", "s", false, "显示所有步骤详情")
	cmd.Flags().BoolP("observations", "o", false, "显示观测结果")

	return cmd
}

// newTraceDeleteCommand 创建删除轨迹的命令
func newTraceDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <trace-id>",
		Short: "删除指定的轨迹",
		Args:  cobra.ExactArgs(1),
		RunE:  runTraceDelete,
	}

	cmd.Flags().BoolP("force", "f", false, "强制删除，不询问确认")

	return cmd
}

// newTraceCleanCommand 创建清理轨迹的命令
func newTraceCleanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "清理旧的轨迹文件",
		RunE:  runTraceClean,
	}

	cmd.Flags().Int("days", 30, "保留多少天内的轨迹")
	cmd.Flags().BoolP("dry-run", "n", false, "预览将被删除的文件，不实际删除")

	return cmd
}

// runTraceList 执行列出轨迹的命令
func runTraceList(cmd *cobra.Command, args []string) error {
	// 获取存储实例
	store, err := getStoreFromConfig(cmd)
	if err != nil {
		return err
	}

	// 获取轨迹列表
	traces, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list traces: %w", err)
	}

	if len(traces) == 0 {
		logger.Info("📭 No traces found")
		return nil
	}

	limit, _ := cmd.Flags().GetInt("limit")
	verbose, _ := cmd.Flags().GetBool("verbose")

	logger.Infof("📋 Found %d trace(s):", len(traces))
	logger.Info("")

	// 显示轨迹列表
	for i, traceFile := range traces {
		if limit > 0 && i >= limit {
			logger.Infof("... and %d more (use --limit to show more)", len(traces)-limit)
			break
		}

		if verbose {
			// 加载轨迹显示详细信息
			trace, err := store.Load(traceFile)
			if err != nil {
				logger.Warnf("⚠️  Failed to load trace %s: %v", traceFile, err)
				continue
			}

			logger.Infof("%d. 📄 %s", i+1, strings.TrimSuffix(traceFile, ".json"))
			logger.Infof("   Goal: %s", truncateString(trace.Goal, 60))
			logger.Infof("   Status: %s | Steps: %d", trace.Status, len(trace.Steps))
			logger.Infof("   Created: %s", trace.CreatedAt.Format("2006-01-02 15:04:05"))
		} else {
			logger.Infof("%d. 📄 %s", i+1, strings.TrimSuffix(traceFile, ".json"))
		}
	}

	return nil
}

// runTraceShow 执行显示轨迹的命令
func runTraceShow(cmd *cobra.Command, args []string) error {
	traceID := args[0]

	// 获取存储实例
	store, err := getStoreFromConfig(cmd)
	if err != nil {
		return err
	}

	// 加载轨迹
	trace, err := store.Load(traceID)
	if err != nil {
		return fmt.Errorf("failed to load trace: %w", err)
	}

	showSteps, _ := cmd.Flags().GetBool("steps")
	showObservations, _ := cmd.Flags().GetBool("observations")

	// 显示基本信息
	logger.Infof("📄 Trace: %s", traceID)
	logger.Info("═══════════════════════════════════════")
	logger.Infof("🎯 Goal: %s", trace.Goal)
	logger.Infof("📊 Status: %s", trace.Status)
	logger.Infof("📈 Steps: %d", len(trace.Steps))
	logger.Infof("⏱️  Created: %s", trace.CreatedAt.Format("2006-01-02 15:04:05"))
	logger.Infof("🔄 Updated: %s", trace.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(trace.Steps) > 0 {
		logger.Info("")
		logger.Info("🔍 Steps:")
		for i, step := range trace.Steps {
			status := "✅"
			if step.Observation != nil && step.Observation.ErrMsg != "" {
				status = "❌"
			}

			logger.Infof("  %d. %s %s", i+1, status, step.Action.Name)

			if step.Action.Reason != "" {
				logger.Infof("     Reason: %s", step.Action.Reason)
			}

			if showSteps && len(step.Action.Args) > 0 {
				logger.Infof("     Args: %+v", step.Action.Args)
			}

			if showObservations && step.Observation != nil {
				if step.Observation.ErrMsg != "" {
					logger.Infof("     Error: %s", step.Observation.ErrMsg)
				} else if step.Observation.Output != nil {
					logger.Infof("     Output: %+v", step.Observation.Output)
				}
				logger.Infof("     Latency: %dms", step.Observation.Latency)
			}
		}
	}

	if len(trace.Reflections) > 0 {
		logger.Info("")
		logger.Info("💭 Reflections:")
		for i, reflection := range trace.Reflections {
			logger.Infof("  %d. Step %d: %s (confidence: %.2f)",
				i+1, reflection.StepIndex, reflection.Result.Reason, reflection.Result.Confidence)
		}
	}

	return nil
}

// runTraceDelete 执行删除轨迹的命令
func runTraceDelete(cmd *cobra.Command, args []string) error {
	traceID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// 获取存储实例
	store, err := getStoreFromConfig(cmd)
	if err != nil {
		return err
	}

	// 如果不是强制删除，询问确认
	if !force {
		logger.Infof("⚠️  Are you sure you want to delete trace '%s'? (y/N): ", traceID)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			logger.Info("Deletion cancelled")
			return nil
		}
	}

	// 删除轨迹
	if err := store.Delete(traceID); err != nil {
		return fmt.Errorf("failed to delete trace: %w", err)
	}

	logger.Infof("🗑️  Trace '%s' deleted successfully", traceID)
	return nil
}

// runTraceClean 执行清理轨迹的命令
func runTraceClean(cmd *cobra.Command, args []string) error {
	days, _ := cmd.Flags().GetInt("days")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// 获取存储实例
	store, err := getStoreFromConfig(cmd)
	if err != nil {
		return err
	}

	// 获取轨迹列表
	traces, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list traces: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var toDelete []string

	// 检查每个轨迹的创建时间
	for _, traceFile := range traces {
		trace, err := store.Load(traceFile)
		if err != nil {
			logger.Warnf("⚠️  Failed to load trace %s: %v", traceFile, err)
			continue
		}

		if trace.CreatedAt.Before(cutoff) {
			toDelete = append(toDelete, traceFile)
		}
	}

	if len(toDelete) == 0 {
		logger.Infof("🧹 No traces older than %d days found", days)
		return nil
	}

	if dryRun {
		logger.Infof("🔍 Would delete %d trace(s) older than %d days:", len(toDelete), days)
		for _, traceFile := range toDelete {
			logger.Infof("  - %s", traceFile)
		}
		return nil
	}

	logger.Infof("🧹 Deleting %d trace(s) older than %d days...", len(toDelete), days)

	deleted := 0
	for _, traceFile := range toDelete {
		if err := store.Delete(traceFile); err != nil {
			logger.Warnf("⚠️  Failed to delete %s: %v", traceFile, err)
		} else {
			deleted++
		}
	}

	logger.Infof("✅ Successfully deleted %d trace(s)", deleted)
	return nil
}

// getStoreFromConfig 从配置创建存储实例
func getStoreFromConfig(cmd *cobra.Command) (state.Store, error) {
	// 获取配置路径
	configPath, _ := cmd.Flags().GetString("config")

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 创建存储实例
	store, err := state.NewStore(&cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return store, nil
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
