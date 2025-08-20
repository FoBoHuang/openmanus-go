package mcp

import (
	"encoding/json"
	"time"
)

// MCPVersion MCP 协议版本
const MCPVersion = "2024-11-05"

// MessageType 消息类型
type MessageType string

const (
	MessageTypeRequest      MessageType = "request"
	MessageTypeResponse     MessageType = "response"
	MessageTypeNotification MessageType = "notification"
)

// Method 方法名称
type Method string

const (
	// 初始化相关
	MethodInitialize  Method = "initialize"
	MethodInitialized Method = "initialized"

	// 工具相关
	MethodListTools Method = "tools/list"
	MethodCallTool  Method = "tools/call"

	// 资源相关
	MethodListResources Method = "resources/list"
	MethodReadResource  Method = "resources/read"

	// 提示相关
	MethodListPrompts Method = "prompts/list"
	MethodGetPrompt   Method = "prompts/get"

	// 日志相关
	MethodSetLogLevel Method = "logging/setLevel"
)

// Message MCP 消息基础结构
type Message struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *string     `json:"id,omitempty"`
	Method  Method      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error MCP 错误结构
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 错误代码常量
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// InitializeParams 初始化参数
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ClientCapabilities     `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
}

// InitializeResult 初始化结果
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ServerCapabilities     `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
	Instructions    string                 `json:"instructions,omitempty"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
}

// ClientCapabilities 客户端能力
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
}

// ServerCapabilities 服务器能力
type ServerCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Logging      *LoggingCapability     `json:"logging,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Tools        *ToolsCapability       `json:"tools,omitempty"`
}

// SamplingCapability 采样能力
type SamplingCapability struct{}

// LoggingCapability 日志能力
type LoggingCapability struct{}

// PromptsCapability 提示能力
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability 资源能力
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapability 工具能力
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientInfo 客户端信息
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool 工具定义
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ListToolsResult 工具列表结果
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams 调用工具参数
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult 调用工具结果
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content 内容
type Content struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// Resource 资源定义
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

// ListResourcesResult 资源列表结果
type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

// ReadResourceParams 读取资源参数
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ReadResourceResult 读取资源结果
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent 资源内容
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

// Prompt 提示定义
type Prompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

// PromptArgument 提示参数
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsResult 提示列表结果
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// GetPromptParams 获取提示参数
type GetPromptParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// GetPromptResult 获取提示结果
type GetPromptResult struct {
	Description string    `json:"description,omitempty"`
	Messages    []Message `json:"messages"`
}

// LogLevel 日志级别
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// SetLogLevelParams 设置日志级别参数
type SetLogLevelParams struct {
	Level LogLevel `json:"level"`
}

// LoggingMessage 日志消息
type LoggingMessage struct {
	Level   LogLevel    `json:"level"`
	Data    interface{} `json:"data"`
	Logger  string      `json:"logger,omitempty"`
	Created time.Time   `json:"created"`
}

// NewRequest 创建请求消息
func NewRequest(id string, method Method, params interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}
}

// NewResponse 创建响应消息
func NewResponse(id string, result interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  result,
	}
}

// NewErrorResponse 创建错误响应消息
func NewErrorResponse(id string, code int, message string, data interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      &id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewNotification 创建通知消息
func NewNotification(method Method, params interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
}

// IsRequest 判断是否为请求消息
func (m *Message) IsRequest() bool {
	return m.ID != nil && m.Method != ""
}

// IsResponse 判断是否为响应消息
func (m *Message) IsResponse() bool {
	return m.ID != nil && m.Method == ""
}

// IsNotification 判断是否为通知消息
func (m *Message) IsNotification() bool {
	return m.ID == nil && m.Method != ""
}

// IsError 判断是否为错误响应
func (m *Message) IsError() bool {
	return m.Error != nil
}

// ToJSON 转换为 JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON 从 JSON 解析
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
