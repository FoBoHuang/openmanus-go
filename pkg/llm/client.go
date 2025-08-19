package llm

import (
	"context"
	"encoding/json"
)

// Message 表示对话消息
type Message struct {
	Role    string `json:"role"`           // system, user, assistant, tool
	Content string `json:"content"`        // 消息内容
	Name    string `json:"name,omitempty"` // 工具名称（用于工具调用结果）
}

// ToolCall 表示工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // 通常是 "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 表示函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// Tool 表示可用工具定义
type Tool struct {
	Type     string       `json:"type"` // 通常是 "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction 表示工具函数定义
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ChatRequest 表示聊天请求
type ChatRequest struct {
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  any       `json:"tool_choice,omitempty"` // "auto", "none", 或具体工具
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatResponse 表示聊天响应
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 表示响应选择
type Choice struct {
	Index        int        `json:"index"`
	Message      Message    `json:"message"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
}

// Usage 表示 token 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Client 定义 LLM 客户端接口
type Client interface {
	// Chat 发送聊天请求
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream 发送流式聊天请求
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, error)

	// GetModel 获取当前使用的模型
	GetModel() string

	// SetModel 设置使用的模型
	SetModel(model string)
}

// Config 表示 LLM 客户端配置
type Config struct {
	Model       string  `json:"model" mapstructure:"model"`
	BaseURL     string  `json:"base_url" mapstructure:"base_url"`
	APIKey      string  `json:"api_key" mapstructure:"api_key"`
	Temperature float64 `json:"temperature" mapstructure:"temperature"`
	MaxTokens   int     `json:"max_tokens" mapstructure:"max_tokens"`
	Timeout     int     `json:"timeout" mapstructure:"timeout"` // 秒
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Model:       "gpt-3.5-turbo",
		BaseURL:     "https://api.openai.com/v1",
		Temperature: 0.1,
		MaxTokens:   4000,
		Timeout:     30,
	}
}

// ParseToolCallArguments 解析工具调用参数
func ParseToolCallArguments(arguments string) (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// CreateToolFromToolInfo 从工具信息创建 LLM Tool
func CreateToolFromToolInfo(name, description string, parameters map[string]any) Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// CreateSystemMessage 创建系统消息
func CreateSystemMessage(content string) Message {
	return Message{
		Role:    "system",
		Content: content,
	}
}

// CreateUserMessage 创建用户消息
func CreateUserMessage(content string) Message {
	return Message{
		Role:    "user",
		Content: content,
	}
}

// CreateAssistantMessage 创建助手消息
func CreateAssistantMessage(content string) Message {
	return Message{
		Role:    "assistant",
		Content: content,
	}
}

// CreateToolMessage 创建工具消息
func CreateToolMessage(toolName, content string) Message {
	return Message{
		Role:    "tool",
		Content: content,
		Name:    toolName,
	}
}
