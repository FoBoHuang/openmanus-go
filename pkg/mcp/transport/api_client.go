package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/mcp"
)

type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type toolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// DeriveMessageURL returns the POST endpoint for JSON-RPC based on either a base URL or an SSE URL.
func DeriveMessageURL(baseOrSSE string) string {
	if strings.HasSuffix(baseOrSSE, "/sse") {
		return strings.TrimSuffix(baseOrSSE, "/sse") + "/message"
	}
	return strings.TrimRight(baseOrSSE, "/") + "/message"
}

// ListTools sends tools/list and waits for the SSE response with the same id.
func ListTools(ctx context.Context, serverName string, cfg config.MCPServerConfig, headers map[string]string) (*mcp.Message, error) {
	reqID := fmt.Sprintf("%s-tools-list-%d", serverName, time.Now().UnixNano())
	payload := jsonrpcRequest{JSONRPC: "2.0", ID: reqID, Method: "tools/list", Params: map[string]any{}}
	body, _ := json.Marshal(payload)

	msgURL := DeriveMessageURL(cfg.URL)
	merged := make(map[string]string)
	for k, v := range cfg.Headers {
		merged[k] = v
	}
	for k, v := range headers {
		merged[k] = v
	}
	if _, respBody, err := PostJSON(ctx, msgURL, body, merged); err != nil {
		return nil, err
	} else if len(respBody) > 0 {
		if msg, err := mcp.FromJSON(respBody); err == nil && msg != nil {
			if msg.ID == nil || *msg.ID == reqID || msg.IsResponse() || msg.IsError() {
				return msg, nil
			}
		}
	}
	return GlobalDispatcher.Wait(ctx, reqID, 30*time.Second)
}

// CallTool sends tools/call and waits for the SSE response with the same id.
func CallTool(ctx context.Context, serverName string, cfg config.MCPServerConfig, toolName string, args map[string]interface{}, headers map[string]string) (*mcp.Message, error) {
	reqID := fmt.Sprintf("%s-tools-call-%d", serverName, time.Now().UnixNano())
	params := toolsCallParams{Name: toolName, Arguments: args}
	payload := jsonrpcRequest{JSONRPC: "2.0", ID: reqID, Method: "tools/call", Params: params}
	body, _ := json.Marshal(payload)

	msgURL := DeriveMessageURL(cfg.URL)
	merged := make(map[string]string)
	for k, v := range cfg.Headers {
		merged[k] = v
	}
	for k, v := range headers {
		merged[k] = v
	}
	if _, respBody, err := PostJSON(ctx, msgURL, body, merged); err != nil {
		return nil, err
	} else if len(respBody) > 0 {
		if msg, err := mcp.FromJSON(respBody); err == nil && msg != nil {
			if msg.ID == nil || *msg.ID == reqID || msg.IsResponse() || msg.IsError() {
				return msg, nil
			}
		}
	}
	return GlobalDispatcher.Wait(ctx, reqID, 30*time.Second)
}
