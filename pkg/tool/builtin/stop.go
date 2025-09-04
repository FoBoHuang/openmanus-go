package builtin

import (
	"context"

	"openmanus-go/pkg/tool"
)

// StopTool 停止执行工具
type StopTool struct {
	*tool.BaseTool
}

// NewStopTool 创建停止执行工具
func NewStopTool() *StopTool {
	return &StopTool{
		BaseTool: tool.NewBaseTool(
			"stop",
			"Stop execution with a reason",
			tool.CreateJSONSchema("object", map[string]any{
				"reason": tool.StringProperty("The reason for stopping"),
			}, []string{"reason"}),
			tool.CreateJSONSchema("object", map[string]any{
				"result":  tool.StringProperty("The stop reason"),
				"stopped": tool.BooleanProperty("Whether execution has stopped"),
			}, []string{"result", "stopped"}),
		),
	}
}

// Invoke 执行停止操作
func (t *StopTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	reason, ok := args["reason"].(string)
	if !ok {
		return map[string]any{
			"error": "reason parameter is required and must be a string",
		}, nil
	}

	return map[string]any{
		"result":  reason,
		"stopped": true,
		"type":    "stop",
	}, nil
}
