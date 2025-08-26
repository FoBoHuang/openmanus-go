package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/mcp/transport"
	"openmanus-go/pkg/tool"
)

// toolInfo describes a tool exposed by an external MCP server
type toolInfo struct {
	Name        string
	Description string
	Properties  map[string]any // best-effort JSON Schema properties
	Required    []string       // best-effort required keys
}

// resolveServer resolves the server name and returns the server config.
// If want is empty and only one server is configured, it auto-selects that one.
// If want is not an exact match, it tries fuzzy name and URL substring matching.
func resolveServer(all map[string]config.MCPServerConfig, want string) (string, config.MCPServerConfig, error) {
	if len(all) == 0 {
		return "", config.MCPServerConfig{}, fmt.Errorf("no MCP servers configured; set [mcp.servers] in config")
	}

	// Auto-select if empty and only one configured
	if want == "" {
		if len(all) == 1 {
			for name, sc := range all {
				return name, sc, nil
			}
		}
		return "", config.MCPServerConfig{}, fmt.Errorf("server is required; set 'server' parameter")
	}

	// Exact match
	if sc, ok := all[want]; ok {
		return want, sc, nil
	}

	// Fuzzy: partial match on name or URL
	lw := strings.ToLower(want)
	for name, sc := range all {
		ln := strings.ToLower(name)
		lu := strings.ToLower(sc.URL)
		if strings.Contains(ln, lw) || strings.Contains(lu, lw) {
			return name, sc, nil
		}
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
	// If server is omitted, auto-pick if only one configured
	var name string
	var serverCfg config.MCPServerConfig
	var err error
	if server == "" {
		name, serverCfg, err = autoSelectServer(t.cfg.MCP.Servers, "")
	} else {
		name, serverCfg, err = resolveServer(t.cfg.MCP.Servers, server)
	}
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
	// If server is omitted, auto-pick if only one configured; otherwise fuzzy/resolve
	var resolvedName string
	var serverCfg config.MCPServerConfig
	var err error
	if server == "" {
		resolvedName, serverCfg, err = autoSelectServer(t.cfg.MCP.Servers, "")
	} else {
		resolvedName, serverCfg, err = resolveServer(t.cfg.MCP.Servers, server)
	}
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
		"server":  tool.StringProperty("MCP server name in config (optional). If omitted, auto-select."),
		"args":    tool.ObjectProperty("Arguments to pass to the selected tool", nil),
		"headers": tool.ObjectProperty("extra HTTP headers (k:v)", nil),
	}, []string{"query"})
	output := tool.CreateJSONSchema("object", map[string]any{
		"selected_tool": tool.StringProperty("Selected MCP tool name"),
		"server_name":   tool.StringProperty("Selected MCP server name"),
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

	// Resolve server: if not provided, attempt auto selection
	var resolvedName string
	var serverCfg config.MCPServerConfig
	var err error
	if server != "" {
		resolvedName, serverCfg, err = resolveServer(t.cfg.MCP.Servers, server)
		if err != nil {
			return nil, err
		}
	} else {
		resolvedName, serverCfg, err = autoSelectServer(t.cfg.MCP.Servers, query)
		if err != nil {
			return nil, err
		}
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
						// Try to capture input schema
						var schema map[string]any
						if s, ok := mm["inputSchema"].(map[string]any); ok {
							schema = s
						} else if s, ok := mm["input_schema"].(map[string]any); ok {
							schema = s
						} else if s, ok := mm["parameters"].(map[string]any); ok {
							schema = s
						} else if s, ok := mm["schema"].(map[string]any); ok {
							schema = s
						}
						if schema != nil {
							if props, ok := schema["properties"].(map[string]any); ok {
								ti.Properties = props
							}
							if req, ok := schema["required"].([]any); ok {
								var reqStr []string
								for _, r := range req {
									if s, ok := r.(string); ok {
										reqStr = append(reqStr, s)
									}
								}
								ti.Required = reqStr
							}
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

	// Find selected tool schema if available
	var selectedToolInfo *toolInfo
	for i := range tools {
		if tools[i].Name == bestName {
			selectedToolInfo = &tools[i]
			break
		}
	}

	// Prepare call args
	var callArgs map[string]interface{}
	if a, ok := args["args"].(map[string]any); ok {
		callArgs = map[string]interface{}{}
		for k, v := range a {
			callArgs[k] = v
		}
	} else {
		callArgs = map[string]interface{}{}
	}

	// If no args provided, inject a sensible default using the natural language query
	if len(callArgs) == 0 {
		// Many stock tools accept 'query' or 'symbol'. We provide 'query' by default.
		callArgs["query"] = query
	}

	// Try to infer market/symbol from query and tool name if missing
	if _, ok := callArgs["symbol"]; !ok {
		mkt, sym := inferMarketAndSymbol(query, bestName)
		if sym != "" {
			callArgs["symbol"] = sym
		}
		if _, okm := callArgs["market"]; !okm && mkt != "" {
			callArgs["market"] = mkt
		}
	}

	// Normalize aliases and enrich by tool name hints
	callArgs = normalizeStockArgs(callArgs, bestName, query)

	// If schema is available, synthesize/rename args to expected keys
	if selectedToolInfo != nil && selectedToolInfo.Properties != nil {
		callArgs = synthesizeArgsFromSchema(selectedToolInfo.Properties, selectedToolInfo.Required, callArgs, query, bestName)
	}

	// Call selected tool
	callMsg, err := transport.CallTool(ctx, resolvedName, serverCfg, bestName, callArgs, headers)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(callMsg)

	// Prepare response including selected server_name
	response := map[string]any{
		"selected_tool": bestName,
		"message":       json.RawMessage(b),
	}
	response["server_name"] = resolvedName

	return response, nil
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

func inferMarketAndSymbol(q string, toolName string) (string, string) {
	// This function is not fully implemented in the original file,
	// but the edit hint implies its existence and potential usage.
	// For now, it will return empty strings.
	return "", ""
}

// normalizeStockArgs maps common aliases and fills defaults guided by tool name
func normalizeStockArgs(args map[string]interface{}, toolName string, q string) map[string]interface{} {
	// Alias mapping: ticker/code -> symbol
	if _, hasSymbol := args["symbol"]; !hasSymbol {
		if v, ok := args["ticker"]; ok {
			args["symbol"] = v
		} else if v, ok := args["code"]; ok {
			args["symbol"] = v
		}
	}

	// If tool targets Hong Kong, ensure market HK unless provided
	lname := strings.ToLower(toolName)
	if _, ok := args["market"]; !ok {
		if strings.Contains(lname, "hongkong") || strings.Contains(lname, "hk") {
			args["market"] = "HK"
		} else if strings.Contains(lname, "america") || strings.Contains(lname, "us") {
			args["market"] = "US"
		} else if strings.Contains(lname, "china-a") || strings.Contains(lname, "a-share") || strings.Contains(lname, "china-a-share") {
			// If symbol hints, pick SH/SZ accordingly later; otherwise leave empty
			if sym, ok := args["symbol"].(string); ok {
				if strings.HasPrefix(sym, "6") || strings.HasPrefix(sym, "688") {
					args["market"] = "SH"
				} else if strings.HasPrefix(sym, "0") || strings.HasPrefix(sym, "3") {
					args["market"] = "SZ"
				}
			}
		}
	}

	// If market absent but symbol looks like specific exchange, set
	if _, ok := args["market"]; !ok {
		if sym, ok := args["symbol"].(string); ok {
			if len(sym) >= 4 && len(sym) <= 5 && regexp.MustCompile(`^\d{4,5}$`).MatchString(sym) {
				args["market"] = "HK"
			} else if strings.HasPrefix(sym, "6") || strings.HasPrefix(sym, "688") || strings.HasPrefix(sym, "601") || strings.HasPrefix(sym, "603") || strings.HasPrefix(sym, "605") {
				args["market"] = "SH"
			} else if strings.HasPrefix(sym, "0") || strings.HasPrefix(sym, "00") || strings.HasPrefix(sym, "002") || strings.HasPrefix(sym, "300") {
				args["market"] = "SZ"
			}
		}
	}

	// Default date to today if tool name suggests price "today" and date missing
	if _, ok := args["date"]; !ok {
		if strings.Contains(strings.ToLower(q), "今日") || strings.Contains(strings.ToLower(q), "today") || strings.Contains(strings.ToLower(toolName), "price") {
			// Use local date YYYY-MM-DD
			// We avoid importing time at top-level previously; add here if needed
			ymd := time.Now().Format("2006-01-02")
			args["date"] = ymd
		}
	}

	return args
}

// synthesizeArgsFromSchema aligns our prepared args to the tool's expected schema keys.
func synthesizeArgsFromSchema(props map[string]any, required []string, args map[string]interface{}, query string, toolName string) map[string]interface{} {
	// Build set of expected keys
	expected := make(map[string]bool)
	for key := range props {
		expected[key] = true
	}

	// Common aliases to expected names
	aliasTo := func(preferred string, aliases ...string) {
		if _, ok := args[preferred]; ok {
			return
		}
		for _, a := range aliases {
			if v, ok := args[a]; ok {
				args[preferred] = v
				break
			}
		}
	}

	// symbol/code/ticker alignment
	if expected["symbol"] {
		aliasTo("symbol", "ticker", "code")
	} else if expected["ticker"] {
		aliasTo("ticker", "symbol", "code")
	} else if expected["code"] {
		aliasTo("code", "symbol", "ticker")
	}

	// market/exchange alignment
	if expected["market"] {
		aliasTo("market", "exchange")
	} else if expected["exchange"] {
		aliasTo("exchange", "market")
	}

	// date/time alignment
	if expected["date"] {
		if _, ok := args["date"]; !ok {
			args["date"] = time.Now().Format("2006-01-02")
		}
	}

	// query/text/keyword alignment
	if expected["query"] {
		if _, ok := args["query"]; !ok {
			args["query"] = query
		}
	} else if expected["text"] {
		aliasTo("text", "query")
		if _, ok := args["text"]; !ok {
			args["text"] = query
		}
	} else if expected["keyword"] {
		aliasTo("keyword", "query")
		if _, ok := args["keyword"]; !ok {
			args["keyword"] = query
		}
	} else if expected["name"] {
		aliasTo("name", "query")
		if _, ok := args["name"]; !ok {
			args["name"] = query
		}
	}

	// Ensure only expected keys if schema is strict (best-effort: do not remove extras here)
	return args
}

// autoSelectServer selects a server from config by:
// 1) If none configured: error
// 2) If one configured: use it
// 3) Otherwise: fuzzy match query against server name and URL; pick highest score
func autoSelectServer(all map[string]config.MCPServerConfig, query string) (string, config.MCPServerConfig, error) {
	if len(all) == 0 {
		return "", config.MCPServerConfig{}, fmt.Errorf("no MCP servers configured; set [mcp.servers] in config or pass 'server'")
	}
	if len(all) == 1 {
		for name, sc := range all {
			return name, sc, nil
		}
	}
	// multiple servers: fuzzy select
	q := strings.ToLower(query)
	bestName := ""
	var bestCfg config.MCPServerConfig
	bestScore := -1
	for name, sc := range all {
		score := 0
		ln := strings.ToLower(name)
		lu := strings.ToLower(sc.URL)
		if strings.Contains(ln, q) {
			score += 5
		}
		if strings.Contains(lu, q) {
			score += 3
		}
		for _, tok := range strings.Fields(q) {
			if len(tok) < 2 {
				continue
			}
			if strings.Contains(ln, tok) {
				score += 2
			}
			if strings.Contains(lu, tok) {
				score += 1
			}
		}
		if score > bestScore {
			bestScore = score
			bestName = name
			bestCfg = sc
		}
	}
	if bestName == "" {
		return "", config.MCPServerConfig{}, fmt.Errorf("failed to select MCP server; pass 'server' explicitly")
	}
	return bestName, bestCfg, nil
}
