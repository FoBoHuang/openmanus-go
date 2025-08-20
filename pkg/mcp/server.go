package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"openmanus-go/pkg/tool"
)

// Server MCP 服务器
type Server struct {
	mu           sync.RWMutex
	toolRegistry *tool.Registry
	capabilities ServerCapabilities
	serverInfo   ServerInfo
	initialized  bool
	logLevel     LogLevel
	logger       *log.Logger
}

// NewServer 创建新的 MCP 服务器
func NewServer(toolRegistry *tool.Registry) *Server {
	return &Server{
		toolRegistry: toolRegistry,
		capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Logging: &LoggingCapability{},
		},
		serverInfo: ServerInfo{
			Name:    "openmanus-go",
			Version: "1.0.0",
		},
		logLevel: LogLevelInfo,
		logger:   log.Default(),
	}
}

// Start 启动 HTTP 服务器
func (s *Server) Start(host string, port int) error {
	mux := http.NewServeMux()

	// 注册路由
	mux.HandleFunc("/", s.handleMCP)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/tools", s.handleToolsList)
	mux.HandleFunc("/tools/invoke", s.handleToolInvoke)

	addr := fmt.Sprintf("%s:%d", host, port)
	s.logger.Printf("Starting MCP server on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

// handleMCP 处理 MCP 协议消息
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		s.writeErrorResponse(w, "", ErrorCodeParseError, "Parse error", err.Error())
		return
	}

	response := s.handleMessage(&msg)
	if response != nil {
		json.NewEncoder(w).Encode(response)
	}
}

// handleMessage 处理 MCP 消息
func (s *Server) handleMessage(msg *Message) *Message {
	if msg.IsRequest() {
		return s.handleRequest(msg)
	} else if msg.IsNotification() {
		s.handleNotification(msg)
		return nil
	}
	return s.newErrorResponse(*msg.ID, ErrorCodeInvalidRequest, "Invalid request", nil)
}

// handleRequest 处理请求消息
func (s *Server) handleRequest(msg *Message) *Message {
	id := *msg.ID

	switch msg.Method {
	case MethodInitialize:
		return s.handleInitialize(id, msg.Params)
	case MethodListTools:
		return s.handleListTools(id)
	case MethodCallTool:
		return s.handleCallTool(id, msg.Params)
	default:
		return s.newErrorResponse(id, ErrorCodeMethodNotFound, "Method not found", string(msg.Method))
	}
}

// handleNotification 处理通知消息
func (s *Server) handleNotification(msg *Message) {
	switch msg.Method {
	case MethodInitialized:
		s.mu.Lock()
		s.initialized = true
		s.mu.Unlock()
		s.logger.Println("Client initialized")
	case MethodSetLogLevel:
		s.handleSetLogLevel(msg.Params)
	}
}

// handleInitialize 处理初始化请求
func (s *Server) handleInitialize(id string, params interface{}) *Message {
	var initParams InitializeParams
	if params != nil {
		data, _ := json.Marshal(params)
		json.Unmarshal(data, &initParams)
	}

	result := InitializeResult{
		ProtocolVersion: MCPVersion,
		Capabilities:    s.capabilities,
		ServerInfo:      s.serverInfo,
		Instructions:    "OpenManus-Go MCP Server - Provides access to various tools and capabilities",
	}

	return NewResponse(id, result)
}

// handleListTools 处理工具列表请求
func (s *Server) handleListTools(id string) *Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0)
	for _, toolName := range s.toolRegistry.ListNames() {
		t, err := s.toolRegistry.Get(toolName)
		if err != nil {
			continue
		}

		mcpTool := Tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
		tools = append(tools, mcpTool)
	}

	result := ListToolsResult{
		Tools: tools,
	}

	return NewResponse(id, result)
}

// handleCallTool 处理工具调用请求
func (s *Server) handleCallTool(id string, params interface{}) *Message {
	var callParams CallToolParams
	if params != nil {
		data, _ := json.Marshal(params)
		if err := json.Unmarshal(data, &callParams); err != nil {
			return s.newErrorResponse(id, ErrorCodeInvalidParams, "Invalid parameters", err.Error())
		}
	}

	// 获取工具
	t, err := s.toolRegistry.Get(callParams.Name)
	if err != nil {
		return s.newErrorResponse(id, ErrorCodeMethodNotFound, "Tool not found", callParams.Name)
	}

	// 调用工具
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := t.Invoke(ctx, callParams.Arguments)

	var result CallToolResult
	if err != nil {
		result = CallToolResult{
			Content: []Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}
	} else {
		// 将输出转换为文本
		outputText := ""
		if output != nil {
			if str, ok := output["result"].(string); ok {
				outputText = str
			} else {
				data, _ := json.MarshalIndent(output, "", "  ")
				outputText = string(data)
			}
		}

		result = CallToolResult{
			Content: []Content{
				{
					Type: "text",
					Text: outputText,
				},
			},
			IsError: false,
		}
	}

	return NewResponse(id, result)
}

// handleSetLogLevel 处理设置日志级别
func (s *Server) handleSetLogLevel(params interface{}) {
	var logParams SetLogLevelParams
	if params != nil {
		data, _ := json.Marshal(params)
		json.Unmarshal(data, &logParams)
		s.mu.Lock()
		s.logLevel = logParams.Level
		s.mu.Unlock()
		s.logger.Printf("Log level set to: %s", logParams.Level)
	}
}

// handleHealth 处理健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"initialized": s.initialized,
		"tools_count": len(s.toolRegistry.ListNames()),
	}

	json.NewEncoder(w).Encode(health)
}

// handleToolsList 处理工具列表 (REST API)
func (s *Server) handleToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	tools := make([]map[string]interface{}, 0)
	for _, toolName := range s.toolRegistry.ListNames() {
		t, err := s.toolRegistry.Get(toolName)
		if err != nil {
			continue
		}

		toolInfo := map[string]interface{}{
			"name":          t.Name(),
			"description":   t.Description(),
			"input_schema":  t.InputSchema(),
			"output_schema": t.OutputSchema(),
		}
		tools = append(tools, toolInfo)
	}

	response := map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	}

	json.NewEncoder(w).Encode(response)
}

// handleToolInvoke 处理工具调用 (REST API)
func (s *Server) handleToolInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var request struct {
		Tool string                 `json:"tool"`
		Args map[string]interface{} `json:"args"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 获取工具
	t, err := s.toolRegistry.Get(request.Tool)
	if err != nil {
		http.Error(w, fmt.Sprintf("Tool not found: %s", request.Tool), http.StatusNotFound)
		return
	}

	// 调用工具
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := t.Invoke(ctx, request.Args)

	response := map[string]interface{}{
		"tool":      request.Tool,
		"success":   err == nil,
		"output":    output,
		"timestamp": time.Now().UTC(),
	}

	if err != nil {
		response["error"] = err.Error()
	}

	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse 写入错误响应
func (s *Server) writeErrorResponse(w http.ResponseWriter, id string, code int, message string, data interface{}) {
	response := s.newErrorResponse(id, code, message, data)
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}

// newErrorResponse 创建错误响应
func (s *Server) newErrorResponse(id string, code int, message string, data interface{}) *Message {
	return NewErrorResponse(id, code, message, data)
}

// SetLogger 设置日志器
func (s *Server) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// GetCapabilities 获取服务器能力
func (s *Server) GetCapabilities() ServerCapabilities {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.capabilities
}

// IsInitialized 检查是否已初始化
func (s *Server) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// GetLogLevel 获取日志级别
func (s *Server) GetLogLevel() LogLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logLevel
}
