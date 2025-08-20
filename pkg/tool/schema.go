package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool 定义工具接口
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]any
	OutputSchema() map[string]any
	Invoke(ctx context.Context, args map[string]any) (map[string]any, error)
}

// ToolInfo 表示工具信息
type ToolInfo struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"input_schema"`
	OutputSchema map[string]any `json:"output_schema"`
}

// BaseTool 提供工具的基础实现
type BaseTool struct {
	name         string
	description  string
	inputSchema  map[string]any
	outputSchema map[string]any
}

// NewBaseTool 创建基础工具
func NewBaseTool(name, description string, inputSchema, outputSchema map[string]any) *BaseTool {
	return &BaseTool{
		name:         name,
		description:  description,
		inputSchema:  inputSchema,
		outputSchema: outputSchema,
	}
}

// Name 返回工具名称
func (bt *BaseTool) Name() string {
	return bt.name
}

// Description 返回工具描述
func (bt *BaseTool) Description() string {
	return bt.description
}

// InputSchema 返回输入 Schema
func (bt *BaseTool) InputSchema() map[string]any {
	return bt.inputSchema
}

// OutputSchema 返回输出 Schema
func (bt *BaseTool) OutputSchema() map[string]any {
	return bt.outputSchema
}

// ValidateInput 验证输入参数
func (bt *BaseTool) ValidateInput(args map[string]any) error {
	// 简单的 JSON Schema 验证
	// 在生产环境中，应该使用更完整的 JSON Schema 验证库
	if bt.inputSchema == nil {
		return nil
	}

	properties, ok := bt.inputSchema["properties"].(map[string]any)
	if !ok {
		return nil
	}

	required, _ := bt.inputSchema["required"].([]string)

	// 检查必需字段
	for _, field := range required {
		if _, exists := args[field]; !exists {
			return fmt.Errorf("required field '%s' is missing", field)
		}
	}

	// 检查字段类型
	for field, value := range args {
		if fieldSchema, exists := properties[field]; exists {
			if err := validateFieldType(field, value, fieldSchema); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateFieldType 验证字段类型
func validateFieldType(field string, value any, schema any) error {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil
	}

	expectedType, ok := schemaMap["type"].(string)
	if !ok {
		return nil
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", field)
		}
	case "number":
		switch value.(type) {
		case int, int64, float64, json.Number:
			// OK
		default:
			return fmt.Errorf("field '%s' must be a number", field)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean", field)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("field '%s' must be an object", field)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("field '%s' must be an array", field)
		}
	}

	return nil
}

// CreateJSONSchema 创建标准的 JSON Schema
func CreateJSONSchema(schemaType string, properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":       schemaType,
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// StringProperty 创建字符串属性
func StringProperty(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

// NumberProperty 创建数字属性
func NumberProperty(description string) map[string]any {
	return map[string]any{
		"type":        "number",
		"description": description,
	}
}

// BooleanProperty 创建布尔属性
func BooleanProperty(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}

// ObjectProperty 创建对象属性
func ObjectProperty(description string, properties map[string]any) map[string]any {
	result := map[string]any{
		"type":        "object",
		"description": description,
	}

	// 只有当 properties 不为 nil 时才添加
	if properties != nil {
		result["properties"] = properties
	}

	return result
}

// ArrayProperty 创建数组属性
func ArrayProperty(description string, itemType map[string]any) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": description,
		"items":       itemType,
	}
}
