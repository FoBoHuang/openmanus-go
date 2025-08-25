package transport

import (
	"context"
	"strings"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/mcp"

	"github.com/r3labs/sse/v2"
)

// MessageHandler 是一个处理 MCP 消息的回调函数类型
type MessageHandler func(msg *mcp.Message)

// SSEClient 是一个通过 SSE 连接的 MCP 客户端
type SSEClient struct {
	config         config.MCPServerConfig
	client         *sse.Client
	messageHandler MessageHandler
	stopChan       chan struct{}
}

// NewSSEClient 创建一个新的 SSE 客户端实例
func NewSSEClient(cfg config.MCPServerConfig, handler MessageHandler) *SSEClient {
	sseURL := ensureSSEPath(cfg.URL)
	cfg.URL = sseURL
	return &SSEClient{
		config:         cfg,
		client:         sse.NewClient(sseURL),
		messageHandler: handler,
		stopChan:       make(chan struct{}),
	}
}

// Start 开始连接并监听 SSE 事件
func (c *SSEClient) Start(ctx context.Context) {
	go c.run(ctx)
	logger.Infof("SSE client started for %s", c.config.URL)
}

// Stop 停止 SSE 客户端
func (c *SSEClient) Stop() {
	close(c.stopChan)
	logger.Infof("SSE client stopped for %s", c.config.URL)
}

func (c *SSEClient) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		default:
			c.connectAndSubscribe(ctx)
			// 如果连接断开，等待一段时间后重连
			logger.Warnf("SSE connection lost for %s. Reconnecting in 5 seconds...", c.config.URL)
			time.Sleep(5 * time.Second)
		}
	}
}

func (c *SSEClient) connectAndSubscribe(ctx context.Context) {
	// sse.Client.Subscribe a new connection every time
	c.client = sse.NewClient(ensureSSEPath(c.config.URL))

	err := c.client.Subscribe("messages", func(event *sse.Event) {
		if len(event.Data) == 0 {
			return
		}
		logger.Debugf("Received SSE event: %s", string(event.Data))

		msg, err := mcp.FromJSON(event.Data)
		if err != nil {
			logger.Errorf("Failed to parse MCP message from SSE event: %v", err)
			return
		}

		if c.messageHandler != nil {
			c.messageHandler(msg)
		}
		GlobalDispatcher.Deliver(msg)
	})

	if err != nil {
		logger.Errorf("Failed to subscribe to SSE stream %s: %v", c.config.URL, err)
	}
}

// Manager 管理多个 SSE 客户端
type Manager struct {
	clients []*SSEClient
}

// NewManager 创建一个新的 SSE 管理器
func NewManager(cfg config.MCPConfig, handler MessageHandler) *Manager {
	mgr := &Manager{}
	for name, serverCfg := range cfg.Servers {
		logger.Infof("Initializing SSE client for %s", name)
		client := NewSSEClient(serverCfg, handler)
		mgr.clients = append(mgr.clients, client)
	}
	return mgr
}

// NewManagerWithFactory 创建一个新的 SSE 管理器，使用每个 serverName 的专属 handler 工厂
func NewManagerWithFactory(cfg config.MCPConfig, handlerFactory func(serverName string) MessageHandler) *Manager {
	mgr := &Manager{}
	for name, serverCfg := range cfg.Servers {
		logger.Infof("Initializing SSE client for %s", name)
		h := handlerFactory(name)
		client := NewSSEClient(serverCfg, h)
		mgr.clients = append(mgr.clients, client)
	}
	return mgr
}

// StartAll 启动所有 SSE 客户端
func (m *Manager) StartAll(ctx context.Context) {
	for _, client := range m.clients {
		client.Start(ctx)
	}
}

// StopAll 停止所有 SSE 客户端
func (m *Manager) StopAll() {
	for _, client := range m.clients {
		client.Stop()
	}
}

// ensureSSEPath appends "/sse" to the base URL if not already present.
func ensureSSEPath(u string) string {
	if u == "" {
		return u
	}
	if strings.HasSuffix(u, "/sse") {
		return u
	}
	if strings.HasSuffix(u, "/") {
		return u + "sse"
	}
	return u + "/sse"
}
