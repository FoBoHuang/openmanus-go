package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/mcp/transport"
	"openmanus-go/pkg/tool"
)

// toolInfo describes a tool exposed by an external MCP server
type toolInfo struct{ Name, Description string }

// resolveServer returns a valid server name and config. If the provided name is empty
// or "default" and exactly one server is configured, it falls back to that single server.
func resolveServer(all map[string]config.MCPServerConfig, want string) (string, config.MCPServerConfig, error) {
	if want != "" && want != "default" {
		if sc, ok := all[want]; ok {
			return want, sc, nil
		}
	}
	if len(all) == 1 {
		for n, sc := range all {
			return n, sc, nil
		}
	}
	if want == "" || want == "default" {
		return "", config.MCPServerConfig{}, fmt.Errorf("server is required; set 'server' or configure a single MCP server")
	}
	return "", config.MCPServerConfig{}, fmt.Errorf("mcp server not found: %s", want)
}

// MCPListToolsTool lists tools from a configured MCP server
type MCPListToolsTool struct {
	*tool.BaseTool
	cfg *config.Config
}

func NewMCPListToolsTool(cfg *config.Config) *MCPListToolsTool {
	input := tool.CreateJSONSchema("object", map[string]any{
		"server":  tool.StringProperty("MCP server name in config"),
		"headers": tool.ObjectProperty("extra HTTP headers (k:v)", nil),
	}, []string{})
	output := tool.CreateJSONSchema("object", map[string]any{
		"message": tool.ObjectProperty("raw MCP response message", nil),
	}, nil)
	return &MCPListToolsTool{BaseTool: tool.NewBaseTool("mcp_list_tools", "List tools from external MCP server", input, output), cfg: cfg}
}

func (t *MCPListToolsTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	server, _ := args["server"].(string)
	headers := map[string]string{}
	if h, ok := args["headers"].(map[string]any); ok {
		for k, v := range h {
			headers[k] = fmt.Sprintf("%v", v)
		}
	}
	name, serverCfg, err := resolveServer(t.cfg.MCP.Servers, server)
	if err != nil {
		return nil, err
	}
	msg, err := transport.ListTools(ctx, name, serverCfg, headers)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(msg)
	return map[string]any{"message": json.RawMessage(b)}, nil
}

// MCPCallTool calls a tool on an external MCP server
type MCPCallTool struct {
	*tool.BaseTool
	cfg *config.Config
}

func NewMCPCallTool(cfg *config.Config) *MCPCallTool {
	input := tool.CreateJSONSchema("object", map[string]any{
		"server":  tool.StringProperty("MCP server name in config"),
		"name":    tool.StringProperty("Tool name to call"),
		"args":    tool.ObjectProperty("Tool arguments", nil),
		"headers": tool.ObjectProperty("extra HTTP headers (k:v)", nil),
	}, []string{"name"})
	output := tool.CreateJSONSchema("object", map[string]any{
		"message": tool.ObjectProperty("raw MCP response message", nil),
	}, nil)
	return &MCPCallTool{BaseTool: tool.NewBaseTool("mcp_call", "Call a tool on external MCP server", input, output), cfg: cfg}
}

func (t *MCPCallTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	server, _ := args["server"].(string)
	nameArg, _ := args["name"].(string)
	if nameArg == "" {
		return nil, fmt.Errorf("name is required")
	}
	var callArgs map[string]interface{}
	if a, ok := args["args"].(map[string]any); ok {
		callArgs = map[string]interface{}{}
		for k, v := range a {
			callArgs[k] = v
		}
	}
	headers := map[string]string{}
	if h, ok := args["headers"].(map[string]any); ok {
		for k, v := range h {
			headers[k] = fmt.Sprintf("%v", v)
		}
	}
	resolvedName, serverCfg, err := resolveServer(t.cfg.MCP.Servers, server)
	if err != nil {
		return nil, err
	}
	msg, err := transport.CallTool(ctx, resolvedName, serverCfg, nameArg, callArgs, headers)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(msg)
	return map[string]any{"message": json.RawMessage(b)}, nil
}

// MCPAutoTool discovers MCP tools on a configured server and calls the best-matching one
// according to a simple fuzzy score against tool name and description.
type MCPAutoTool struct {
	*tool.BaseTool
	cfg *config.Config
}

func NewMCPAutoTool(cfg *config.Config) *MCPAutoTool {
	input := tool.CreateJSONSchema("object", map[string]any{
		"query":   tool.StringProperty("Intent or task, used to select MCP tool"),
		"server":  tool.StringProperty("MCP server name in config"),
		"args":    tool.ObjectProperty("Arguments to pass to the selected tool", nil),
		"headers": tool.ObjectProperty("extra HTTP headers (k:v)", nil),
	}, []string{"query"})
	output := tool.CreateJSONSchema("object", map[string]any{
		"selected_tool": tool.StringProperty("Selected MCP tool name"),
		"message":       tool.ObjectProperty("raw MCP response message", nil),
	}, nil)
	return &MCPAutoTool{BaseTool: tool.NewBaseTool("mcp_auto", "Auto-select and call a tool exposed by an external MCP server based on the query", input, output), cfg: cfg}
}

func (t *MCPAutoTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	server, _ := args["server"].(string)

	headers := map[string]string{}
	if h, ok := args["headers"].(map[string]any); ok {
		for k, v := range h {
			headers[k] = fmt.Sprintf("%v", v)
		}
	}

	resolvedName, serverCfg, err := resolveServer(t.cfg.MCP.Servers, server)
	if err != nil {
		return nil, err
	}

	// List tools
	listMsg, err := transport.ListTools(ctx, resolvedName, serverCfg, headers)
	if err != nil {
		return nil, err
	}

	// Extract tools from message.result.tools (best-effort schema)
	tools := make([]toolInfo, 0)
	if listMsg != nil && listMsg.Result != nil {
		if m, ok := listMsg.Result.(map[string]any); ok {
			if arr, ok := m["tools"].([]any); ok {
				for _, it := range arr {
					if mm, ok := it.(map[string]any); ok {
						ti := toolInfo{}
						if v, ok := mm["name"].(string); ok {
							ti.Name = v
						}
						if v, ok := mm["description"].(string); ok {
							ti.Description = v
						}
						if ti.Name != "" {
							tools = append(tools, ti)
						}
					}
				}
			}
		}
	}
	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools available on MCP server %s", resolvedName)
	}

	// Select best tool by simple fuzzy score
	bestName := tools[0].Name
	bestScore := scoreMatch(query, tools[0])
	for i := 1; i < len(tools); i++ {
		s := scoreMatch(query, tools[i])
		if s > bestScore {
			bestScore = s
			bestName = tools[i].Name
		}
	}

	// Prepare call args
	var callArgs map[string]interface{}
	if a, ok := args["args"].(map[string]any); ok {
		callArgs = map[string]interface{}{}
		for k, v := range a {
			callArgs[k] = v
		}
	}

	// Call selected tool
	callMsg, err := transport.CallTool(ctx, resolvedName, serverCfg, bestName, callArgs, headers)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(callMsg)
	return map[string]any{"selected_tool": bestName, "message": json.RawMessage(b)}, nil
}

// scoreMatch computes a simple score using substring checks and token overlap
func scoreMatch(query string, t toolInfo) int {
	q := strings.ToLower(query)
	name := strings.ToLower(t.Name)
	desc := strings.ToLower(t.Description)
	score := 0
	if strings.Contains(name, q) {
		score += 5
	}
	if strings.Contains(desc, q) {
		score += 3
	}
	// token overlap
	for _, tok := range strings.Fields(q) {
		if len(tok) < 2 {
			continue
		}
		if strings.Contains(name, tok) {
			score += 2
		}
		if strings.Contains(desc, tok) {
			score += 1
		}
	}
	return score
}
