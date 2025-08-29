package commands

import (
	"encoding/json"
	"fmt"

	"openmanus-go/pkg/logger"
	"openmanus-go/pkg/mcp"
	"openmanus-go/pkg/mcp/transport"
)

// BuildMCPMessageHandler builds a handler that converts MCP notifications into
// actionable analysis goals and forwards them to the provided channel.
func BuildMCPMessageHandler(mcpEvents chan<- string) func(*mcp.Message) {
	return func(msg *mcp.Message) {
		// Log raw message for debugging (skip ping messages to reduce noise)
		if msg.Method != "ping" {
			if b, err := json.Marshal(msg); err == nil {
				logger.Debugf("[MCP] Message: %s", string(b))
			}
		}

		if !msg.IsNotification() {
			return
		}

		// Try to extract a stock symbol from common keys in params
		var symbol string
		if m, ok := msg.Params.(map[string]interface{}); ok {
			if v, ok := m["symbol"]; ok {
				symbol = fmt.Sprintf("%v", v)
			} else if v, ok := m["code"]; ok {
				symbol = fmt.Sprintf("%v", v)
			} else if v, ok := m["ticker"]; ok {
				symbol = fmt.Sprintf("%v", v)
			}
		}

		// Construct a rich analysis goal to guide the agent
		var goal string
		if symbol != "" {
			goal = fmt.Sprintf(
				"请作为资深金融分析师，对股票 %s 进行系统分析：\n"+
					"1) 最近价格相对 30/90/240 日均线；\n"+
					"2) K线与成交量形态，是否存在趋势或背离；\n"+
					"3) 行业与新闻情绪（雪球/东方财富/同花顺/财报源）；\n"+
					"4) 核心风险点与投资结论（明确看法与理由）。\n"+
					"注意：优先使用 http_client 或 browser 访问权威来源；避免重复调用同一工具超过 3 次；输出结构化报告（概要、数据要点、结论）。",
				symbol,
			)
		} else {
			goal = fmt.Sprintf(
				"收到事件 %s，请进行与股票相关的完整分析，包含均线、形态、新闻情绪与结论。优先使用 http_client/browser 获取权威数据，避免重复调用同一工具超过 3 次。",
				msg.Method,
			)
		}

		logger.Infof("[MCP] Sending new goal to agent: %s", goal)
		mcpEvents <- goal
	}
}

// BuildServerAwareMCPHandlerFactory returns a factory that builds per-server handlers.
// You can plug specialized parsing/mapping rules by serverName and optionally method.
func BuildServerAwareMCPHandlerFactory(mcpEvents chan<- string) func(serverName string) transport.MessageHandler {
	return func(serverName string) transport.MessageHandler {
		// Choose specialized handler by serverName
		switch serverName {
		case "mcp-stock-helper":
			return transport.MessageHandler(func(msg *mcp.Message) {
				// Log non-ping messages only to reduce noise
				if msg.Method != "ping" {
					if b, err := json.Marshal(msg); err == nil {
						logger.Debugf("[MCP %s] %s", serverName, string(b))
					}
				}
				if !msg.IsNotification() {
					transport.GlobalDispatcher.Deliver(msg)
					return
				}

				// Example of method-specific tuning
				switch string(msg.Method) {
				case "stock/update", "stock/alert":
					// Fallthrough to default mapping below
				}

				// Default: delegate to generic mapper
				BuildMCPMessageHandler(mcpEvents)(msg)
			})
		default:
			// Unknown server: fallback to generic handler
			return transport.MessageHandler(BuildMCPMessageHandler(mcpEvents))
		}
	}
}
