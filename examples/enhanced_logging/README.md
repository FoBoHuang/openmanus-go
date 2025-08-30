# 增强的日志输出

本示例展示了改进后的日志输出，让您能够清晰地了解：

## 主要改进

### 1. 🛠️ 清晰的工具使用显示

**规划阶段：**
- `🛠️ [LLM] LLM decided to use a tool` - 明确显示LLM选择使用工具
- `🎯 [TOOL] Selected: 🌐 tool_name (MCP tool)` - 显示选择的工具和类型
- `📡 [SERVER] From MCP server: server_name` - 显示MCP服务器信息
- `⚙️ [ARGS] Tool arguments: {...}` - 显示工具参数

**或者：**
- `💭 [LLM] LLM decided not to use any tools` - 明确显示LLM选择不使用工具
- `📝 [RESPONSE] LLM response: ...` - 显示LLM的直接回答

### 2. 📊 详细的执行进度显示

**步骤进度：**
```
🤔 [STEP 1/10] Planning next action...
⏱️ [PROGRESS] 0.0% complete | Elapsed: 2s
```

**工具执行：**
```
🔧 [TOOL] Executing 🌐 stock_query (MCP tool)
📡 [SERVER] Calling MCP server: stock_server
✅ [RESULT] stock_query completed successfully (1250ms)
📄 [OUTPUT] {"stock": "HK1168", "price": 5.67, "currency": "HKD"}
```

### 3. 🏁 完整的执行总结

```
═══════════════════════════════════════════════════════════════
🏁 [DONE] Execution completed!
📋 [SUMMARY] Goal: 查看港股西部水泥的股价是多少,并将结果保存到workspace目录下
📊 [STATS] Steps: 3/10 | Status: completed | Duration: 15s
🔍 [STEPS] Execution trace:
   1. ✅ stock_query
   2. ✅ fs
   3. ✅ direct_answer
═══════════════════════════════════════════════════════════════
```

## 工具类型标识

- 🔧 **内置工具** (Built-in tools): 如文件系统、HTTP客户端等
- 🌐 **MCP工具** (MCP tools): 来自外部MCP服务器的工具

## 日志级别

- **INFO**: 主要执行流程和结果
- **DEBUG**: 详细的工具参数和输出
- **WARN**: 错误和警告信息

## 示例命令

```bash
# 运行带有详细日志的命令
./bin/openmanus run "查看港股西部水泥的股价是多少,并将结果保存到workspace目录下"
```

## 预期日志输出

```
🚀 [AGENT] Starting execution: 查看港股西部水泥的股价是多少,并将结果保存到workspace目录下
📊 [BUDGET] Max steps: 10 | Max tokens: 8000 | Max duration: 5m0s
═══════════════════════════════════════════════════════════════

🤔 [STEP 1/10] Planning next action...
⏱️ [PROGRESS] 0.0% complete | Elapsed: 0s
🛠️ [LLM] LLM decided to use a tool
🎯 [TOOL] Selected: 🌐 stock_query (MCP tool)
📡 [SERVER] From MCP server: stock_server
⚙️ [ARGS] Tool arguments: {symbol: "1168.HK", market: "HK"}
⚡ [EXEC] Executing stock_query...
🔧 [TOOL] Executing 🌐 stock_query (MCP tool)
📡 [SERVER] Calling MCP server: stock_server
✅ [RESULT] stock_query completed successfully (1250ms)
📄 [OUTPUT] {"stock": "1168.HK", "name": "西部水泥", "price": 5.67, "currency": "HKD", "change": "+0.12"}

🤔 [STEP 2/10] Planning next action...
⏱️ [PROGRESS] 10.0% complete | Elapsed: 3s
🛠️ [LLM] LLM decided to use a tool
🎯 [TOOL] Selected: 🔧 fs (Built-in tool)
⚙️ [ARGS] Tool arguments: {operation: "write", path: "workspace/stock_result.json", content: "..."}
⚡ [EXEC] Executing fs...
🔧 [TOOL] Executing 🔧 fs (Built-in tool)
✅ [RESULT] fs completed successfully (45ms)
📄 [OUTPUT] {"success": true, "path": "workspace/stock_result.json"}

🤔 [STEP 3/10] Planning next action...
⏱️ [PROGRESS] 20.0% complete | Elapsed: 5s
💭 [LLM] LLM decided not to use any tools
📝 [RESPONSE] 已成功查询港股西部水泥(1168.HK)的股价为5.67港币，较前日上涨0.12港币...
✅ [ANSWER] Task verified as complete (confidence: 0.9)

═══════════════════════════════════════════════════════════════
🏁 [DONE] Execution completed!
📋 [SUMMARY] Goal: 查看港股西部水泥的股价是多少,并将结果保存到workspace目录下
📊 [STATS] Steps: 3/10 | Status: completed | Duration: 15s
🔍 [STEPS] Execution trace:
   1. ✅ stock_query
   2. ✅ fs
   3. ✅ direct_answer
═══════════════════════════════════════════════════════════════
```

这样的日志输出让您能够：

1. **清楚地看到LLM是否使用了工具**
2. **知道使用了什么类型的工具**（内置 vs MCP）
3. **了解当前执行进度**（步骤数、百分比、耗时）
4. **看到工具执行的详细信息**（参数、结果、耗时）
5. **获得完整的执行总结**（成功/失败状态、步骤轨迹）
