package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// 推荐服务工具定义 —— 根据你的实际接口修改 inputSchema
var recommendTools = []map[string]interface{}{
	{
		"name":        "find_person_for_male_user",
		"description": "为男性用户推荐匹配的人选。根据用户画像和偏好，从推荐服务获取推荐列表。",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_id": map[string]interface{}{
					"type":        "string",
					"description": "男性用户的唯一标识ID",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回推荐人数上限，默认10",
				},
				// 根据实际接口需要，在这里添加更多参数
				// "age_min":   {"type": "integer", "description": "最小年龄"},
				// "age_max":   {"type": "integer", "description": "最大年龄"},
				// "city":      {"type": "string",  "description": "目标城市"},
			},
			"required": []string{"user_id"},
		},
	},
}

type server struct {
	apiBase    string
	httpClient *http.Client
}

func newServer(apiBase string) *server {
	return &server{
		apiBase: apiBase,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *server) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeRPCError(w, nil, -32700, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req jsonrpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeRPCError(w, nil, -32700, "invalid JSON")
		return
	}

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "tools/list":
		s.handleToolsList(w, req)
	case "tools/call":
		s.handleToolsCall(w, req)
	default:
		writeRPCError(w, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *server) handleInitialize(w http.ResponseWriter, req jsonrpcRequest) {
	writeRPCResult(w, req.ID, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mcp-recommend-server",
			"version": "1.0.0",
		},
	})
}

func (s *server) handleToolsList(w http.ResponseWriter, req jsonrpcRequest) {
	writeRPCResult(w, req.ID, map[string]interface{}{
		"tools": recommendTools,
	})
}

func (s *server) handleToolsCall(w http.ResponseWriter, req jsonrpcRequest) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeRPCError(w, req.ID, -32602, "invalid params")
		return
	}

	if params.Name != "find_person_for_male_user" {
		writeRPCError(w, req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name))
		return
	}

	result, err := s.callRecommendAPI(params.Arguments)
	if err != nil {
		writeRPCError(w, req.ID, -32603, fmt.Sprintf("recommend API error: %v", err))
		return
	}

	writeRPCResult(w, req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": result},
		},
	})
}

// callRecommendAPI 调用实际的推荐服务接口
// 你需要根据实际的接口协议调整这个方法：请求方式、路径、参数格式、响应解析等
func (s *server) callRecommendAPI(args map[string]interface{}) (string, error) {
	url := fmt.Sprintf("%s/api_server/xxx", s.apiBase)

	payload, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("marshal args: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

func writeRPCResult(w http.ResponseWriter, id json.RawMessage, result interface{}) {
	resp := jsonrpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	resp := jsonrpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	port := flag.Int("port", 9100, "MCP server 监听端口")
	apiBase := flag.String("api-base", "http://localhost:8080", "推荐服务 API 的基础地址（不带路径）")
	flag.Parse()

	srv := newServer(*apiBase)

	mux := http.NewServeMux()
	mux.HandleFunc("/message", srv.handleMessage)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("[mcp-recommend-server] listening on :%d, forwarding to %s", *port, *apiBase)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[mcp-recommend-server] shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}
