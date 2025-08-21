package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"openmanus-go/pkg/logger"
)

// OpenAIClient OpenAI 兼容的客户端实现
type OpenAIClient struct {
	config     *Config
	httpClient *http.Client
}

// NewOpenAIClient 创建 OpenAI 客户端
func NewOpenAIClient(config *Config) *OpenAIClient {
	if config == nil {
		config = DefaultConfig()
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &OpenAIClient{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Chat 发送聊天请求
func (c *OpenAIClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 设置默认参数
	if req.Model == "" {
		req.Model = c.config.Model
	}
	if req.Temperature == 0 && c.config.Temperature > 0 {
		req.Temperature = c.config.Temperature
	}
	if req.MaxTokens == 0 && c.config.MaxTokens > 0 {
		req.MaxTokens = c.config.MaxTokens
	}

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.Debugw("llm.chat.request", "model", req.Model, "messages", len(req.Messages), "tools", len(req.Tools), "temperature", req.Temperature, "max_tokens", req.MaxTokens)

	// 创建 HTTP 请求
	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	httpReq.Header.Set("User-Agent", "OpenManus-Go/1.0")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.Errorw("llm.chat.transport_error", "error", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error.Message != "" {
			logger.Errorw("llm.chat.api_error", "status", resp.StatusCode, "message", errorResp.Error.Message)
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}

		logger.Errorw("llm.chat.api_error_raw", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.Debugw("llm.chat.response", "id", chatResp.ID, "choices", len(chatResp.Choices), "usage", chatResp.Usage)

	return &chatResp, nil
}

// ChatStream 发送流式聊天请求
func (c *OpenAIClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, error) {
	// 设置流式请求
	req.Stream = true

	// 设置默认参数
	if req.Model == "" {
		req.Model = c.config.Model
	}
	if req.Temperature == 0 && c.config.Temperature > 0 {
		req.Temperature = c.config.Temperature
	}
	if req.MaxTokens == 0 && c.config.MaxTokens > 0 {
		req.MaxTokens = c.config.MaxTokens
	}

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	httpReq.Header.Set("User-Agent", "OpenManus-Go/1.0")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	// 创建响应通道
	respChan := make(chan *ChatResponse, 10)

	// 启动协程处理流式响应
	go func() {
		defer close(respChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var streamResp struct {
				ChatResponse
				Object string `json:"object"`
			}

			if err := decoder.Decode(&streamResp); err != nil {
				if err == io.EOF {
					break
				}
				// 流式响应可能包含非 JSON 数据，跳过错误
				continue
			}

			select {
			case respChan <- &streamResp.ChatResponse:
			case <-ctx.Done():
				return
			}

			// 检查是否完成
			if len(streamResp.Choices) > 0 && streamResp.Choices[0].FinishReason != "" {
				break
			}
		}
	}()

	return respChan, nil
}

// GetModel 获取当前模型
func (c *OpenAIClient) GetModel() string {
	return c.config.Model
}

// SetModel 设置模型
func (c *OpenAIClient) SetModel(model string) {
	c.config.Model = model
}

// SetAPIKey 设置 API Key
func (c *OpenAIClient) SetAPIKey(apiKey string) {
	c.config.APIKey = apiKey
}

// SetBaseURL 设置 Base URL
func (c *OpenAIClient) SetBaseURL(baseURL string) {
	c.config.BaseURL = baseURL
}

// SetTemperature 设置温度
func (c *OpenAIClient) SetTemperature(temperature float64) {
	c.config.Temperature = temperature
}

// SetMaxTokens 设置最大 token 数
func (c *OpenAIClient) SetMaxTokens(maxTokens int) {
	c.config.MaxTokens = maxTokens
}

// SetTimeout 设置超时时间
func (c *OpenAIClient) SetTimeout(timeout time.Duration) {
	c.config.Timeout = int(timeout.Seconds())
	c.httpClient.Timeout = timeout
}

// GetConfig 获取配置
func (c *OpenAIClient) GetConfig() *Config {
	// 返回配置副本
	configCopy := *c.config
	return &configCopy
}
