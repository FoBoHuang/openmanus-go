package builtin

import (
	"context"

	"openmanus-go/pkg/tool"
)

// DirectAnswerTool 直接回答工具
type DirectAnswerTool struct {
	*tool.BaseTool
}

// NewDirectAnswerTool 创建直接回答工具
func NewDirectAnswerTool() *DirectAnswerTool {
	return &DirectAnswerTool{
		BaseTool: tool.NewBaseTool(
			"direct_answer",
			"Provide a direct answer to the user's question without using any tools",
			tool.CreateJSONSchema("object", map[string]any{
				"answer": tool.StringProperty("The direct answer to provide to the user"),
			}, []string{"answer"}),
			tool.CreateJSONSchema("object", map[string]any{
				"result": tool.StringProperty("The provided answer"),
			}, []string{"result"}),
		),
	}
}

// Invoke 执行直接回答
func (t *DirectAnswerTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	answer, ok := args["answer"].(string)
	if !ok {
		return map[string]any{
			"error": "answer parameter is required and must be a string",
		}, nil
	}

	return map[string]any{
		"result": answer,
		"type":   "direct_answer",
	}, nil
}
