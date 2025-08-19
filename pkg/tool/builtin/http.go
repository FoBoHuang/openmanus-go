package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"openmanus-go/pkg/tool"
)

// HTTPTool HTTP 请求工具
type HTTPTool struct {
	*tool.BaseTool
	client *http.Client
}

// NewHTTPTool 创建 HTTP 工具
func NewHTTPTool() *HTTPTool {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"url":    tool.StringProperty("请求的 URL"),
		"method": tool.StringProperty("HTTP 方法 (GET, POST, PUT, DELETE 等)"),
		"headers": tool.ObjectProperty("请求头", map[string]any{
			"additionalProperties": tool.StringProperty("请求头值"),
		}),
		"body":    tool.StringProperty("请求体内容"),
		"timeout": tool.NumberProperty("超时时间（秒）"),
	}, []string{"url"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"status_code":  tool.NumberProperty("HTTP 状态码"),
		"headers":      tool.ObjectProperty("响应头", nil),
		"body":         tool.StringProperty("响应体内容"),
		"content_type": tool.StringProperty("内容类型"),
		"size":         tool.NumberProperty("响应体大小（字节）"),
	}, []string{"status_code", "body"})

	baseTool := tool.NewBaseTool(
		"http",
		"发送 HTTP 请求，支持 GET、POST、PUT、DELETE 等方法",
		inputSchema,
		outputSchema,
	)

	return &HTTPTool{
		BaseTool: baseTool,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Invoke 执行 HTTP 请求
func (h *HTTPTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	// 解析参数
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required and must be a string")
	}

	method, ok := args["method"].(string)
	if !ok {
		method = "GET"
	}
	method = strings.ToUpper(method)

	// 构建请求体
	var body io.Reader
	if bodyStr, ok := args["body"].(string); ok && bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	if headers, ok := args["headers"].(map[string]any); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, valueStr)
			}
		}
	}

	// 设置默认 User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "OpenManus-Go/1.0")
	}

	// 设置超时
	client := h.client
	if timeoutSec, ok := args["timeout"].(float64); ok && timeoutSec > 0 {
		client = &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		}
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 构建响应头映射
	headers := make(map[string]any)
	for key, values := range resp.Header {
		if len(values) == 1 {
			headers[key] = values[0]
		} else {
			headers[key] = values
		}
	}

	// 构建结果
	result := map[string]any{
		"status_code":  resp.StatusCode,
		"headers":      headers,
		"body":         string(respBody),
		"content_type": resp.Header.Get("Content-Type"),
		"size":         len(respBody),
	}

	return result, nil
}

// HTTPClientTool 高级 HTTP 客户端工具
type HTTPClientTool struct {
	*tool.BaseTool
	client *http.Client
}

// NewHTTPClientTool 创建高级 HTTP 客户端工具
func NewHTTPClientTool() *HTTPClientTool {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"requests": tool.ArrayProperty("HTTP 请求列表", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":     tool.StringProperty("请求 URL"),
				"method":  tool.StringProperty("HTTP 方法"),
				"headers": tool.ObjectProperty("请求头", nil),
				"body":    tool.StringProperty("请求体"),
				"name":    tool.StringProperty("请求名称（可选）"),
			},
			"required": []string{"url"},
		}),
		"concurrent": tool.BooleanProperty("是否并发执行"),
		"timeout":    tool.NumberProperty("超时时间（秒）"),
	}, []string{"requests"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"results": tool.ArrayProperty("请求结果列表", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        tool.StringProperty("请求名称"),
				"status_code": tool.NumberProperty("状态码"),
				"body":        tool.StringProperty("响应体"),
				"error":       tool.StringProperty("错误信息"),
			},
		}),
		"total_time": tool.NumberProperty("总执行时间（毫秒）"),
	}, []string{"results"})

	baseTool := tool.NewBaseTool(
		"http_client",
		"高级 HTTP 客户端，支持批量请求和并发执行",
		inputSchema,
		outputSchema,
	)

	return &HTTPClientTool{
		BaseTool: baseTool,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Invoke 执行批量 HTTP 请求
func (hc *HTTPClientTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	start := time.Now()

	// 解析请求列表
	requestsRaw, ok := args["requests"].([]any)
	if !ok {
		return nil, fmt.Errorf("requests must be an array")
	}

	concurrent, _ := args["concurrent"].(bool)
	timeout, _ := args["timeout"].(float64)

	// 设置客户端超时
	client := hc.client
	if timeout > 0 {
		client = &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		}
	}

	var results []map[string]any

	if concurrent {
		// 并发执行
		results = hc.executeConcurrent(ctx, client, requestsRaw)
	} else {
		// 顺序执行
		results = hc.executeSequential(ctx, client, requestsRaw)
	}

	totalTime := time.Since(start)

	return map[string]any{
		"results":    results,
		"total_time": totalTime.Milliseconds(),
	}, nil
}

// executeSequential 顺序执行请求
func (hc *HTTPClientTool) executeSequential(ctx context.Context, client *http.Client, requests []any) []map[string]any {
	var results []map[string]any

	for i, reqRaw := range requests {
		reqMap, ok := reqRaw.(map[string]any)
		if !ok {
			results = append(results, map[string]any{
				"name":  fmt.Sprintf("request_%d", i),
				"error": "invalid request format",
			})
			continue
		}

		result := hc.executeRequest(ctx, client, reqMap, i)
		results = append(results, result)
	}

	return results
}

// executeConcurrent 并发执行请求
func (hc *HTTPClientTool) executeConcurrent(ctx context.Context, client *http.Client, requests []any) []map[string]any {
	results := make([]map[string]any, len(requests))

	type result struct {
		index int
		data  map[string]any
	}

	resultChan := make(chan result, len(requests))

	// 启动并发请求
	for i, reqRaw := range requests {
		go func(idx int, req any) {
			reqMap, ok := req.(map[string]any)
			if !ok {
				resultChan <- result{
					index: idx,
					data: map[string]any{
						"name":  fmt.Sprintf("request_%d", idx),
						"error": "invalid request format",
					},
				}
				return
			}

			data := hc.executeRequest(ctx, client, reqMap, idx)
			resultChan <- result{index: idx, data: data}
		}(i, reqRaw)
	}

	// 收集结果
	for i := 0; i < len(requests); i++ {
		res := <-resultChan
		results[res.index] = res.data
	}

	return results
}

// executeRequest 执行单个请求
func (hc *HTTPClientTool) executeRequest(ctx context.Context, client *http.Client, reqMap map[string]any, index int) map[string]any {
	url, _ := reqMap["url"].(string)
	method, _ := reqMap["method"].(string)
	if method == "" {
		method = "GET"
	}

	name, _ := reqMap["name"].(string)
	if name == "" {
		name = fmt.Sprintf("request_%d", index)
	}

	// 构建请求体
	var body io.Reader
	if bodyStr, ok := reqMap["body"].(string); ok && bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, body)
	if err != nil {
		return map[string]any{
			"name":  name,
			"error": fmt.Sprintf("failed to create request: %v", err),
		}
	}

	// 设置请求头
	if headers, ok := reqMap["headers"].(map[string]any); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, valueStr)
			}
		}
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return map[string]any{
			"name":  name,
			"error": fmt.Sprintf("request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]any{
			"name":        name,
			"status_code": resp.StatusCode,
			"error":       fmt.Sprintf("failed to read response: %v", err),
		}
	}

	return map[string]any{
		"name":        name,
		"status_code": resp.StatusCode,
		"body":        string(respBody),
	}
}
