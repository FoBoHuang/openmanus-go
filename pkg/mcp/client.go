package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client MCP 客户端
type Client struct {
	mu           sync.RWMutex
	baseURL      string
	httpClient   *http.Client
	capabilities ClientCapabilities
	clientInfo   ClientInfo
	initialized  bool
	serverInfo   *ServerInfo
	tools        []Tool
}

// NewClient 创建新的 MCP 客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		capabilities: ClientCapabilities{},
		clientInfo: ClientInfo{
			Name:    "openmanus-go-client",
			Version: "1.0.0",
		},
	}
}

// Initialize 初始化客户端连接
func (c *Client) Initialize(ctx context.Context) error {
	params := InitializeParams{
		ProtocolVersion: MCPVersion,
		Capabilities:    c.capabilities,
		ClientInfo:      c.clientInfo,
	}

	request := NewRequest("init", MethodInitialize, params)
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	if response.IsError() {
		return fmt.Errorf("initialize failed: %s", response.Error.Message)
	}

	var result InitializeResult
	if err := c.unmarshalResult(response.Result, &result); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	c.mu.Lock()
	c.serverInfo = &result.ServerInfo
	c.initialized = true
	c.mu.Unlock()

	// 发送初始化完成通知
	notification := NewNotification(MethodInitialized, nil)
	_, err = c.sendRequest(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// ListTools 获取工具列表
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	if !c.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	request := NewRequest("list_tools", MethodListTools, nil)
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send list tools request: %w", err)
	}

	if response.IsError() {
		return nil, fmt.Errorf("list tools failed: %s", response.Error.Message)
	}

	var result ListToolsResult
	if err := c.unmarshalResult(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list tools result: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	return result.Tools, nil
}

// CallTool 调用工具
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error) {
	if !c.IsInitialized() {
		return nil, fmt.Errorf("client not initialized")
	}

	params := CallToolParams{
		Name:      name,
		Arguments: arguments,
	}

	request := NewRequest("call_tool", MethodCallTool, params)
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send call tool request: %w", err)
	}

	if response.IsError() {
		return nil, fmt.Errorf("call tool failed: %s", response.Error.Message)
	}

	var result CallToolResult
	if err := c.unmarshalResult(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call tool result: %w", err)
	}

	return &result, nil
}

// CallToolHTTP 通过 HTTP REST API 调用工具
func (c *Client) CallToolHTTP(ctx context.Context, name string, arguments map[string]interface{}) (map[string]interface{}, error) {
	url := c.baseURL + "/tools/invoke"

	requestBody := map[string]interface{}{
		"tool": name,
		"args": arguments,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ListToolsHTTP 通过 HTTP REST API 获取工具列表
func (c *Client) ListToolsHTTP(ctx context.Context) ([]map[string]interface{}, error) {
	url := c.baseURL + "/tools"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
		Count int                      `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Tools, nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	url := c.baseURL + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// SetLogLevel 设置日志级别
func (c *Client) SetLogLevel(ctx context.Context, level LogLevel) error {
	if !c.IsInitialized() {
		return fmt.Errorf("client not initialized")
	}

	params := SetLogLevelParams{
		Level: level,
	}

	notification := NewNotification(MethodSetLogLevel, params)
	_, err := c.sendRequest(ctx, notification)
	return err
}

// sendRequest 发送请求
func (c *Client) sendRequest(ctx context.Context, message *Message) (*Message, error) {
	url := c.baseURL + "/"

	jsonData, err := message.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// 对于通知消息，可能没有响应
	if message.IsNotification() {
		return nil, nil
	}

	response, err := FromJSON(body)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

// unmarshalResult 解析结果
func (c *Client) unmarshalResult(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// IsInitialized 检查是否已初始化
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// GetServerInfo 获取服务器信息
func (c *Client) GetServerInfo() *ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// GetTools 获取缓存的工具列表
func (c *Client) GetTools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// SetHTTPClient 设置 HTTP 客户端
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetCapabilities 设置客户端能力
func (c *Client) SetCapabilities(capabilities ClientCapabilities) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.capabilities = capabilities
}

// SetClientInfo 设置客户端信息
func (c *Client) SetClientInfo(info ClientInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clientInfo = info
}
