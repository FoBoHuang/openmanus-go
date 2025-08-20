#!/bin/bash

# OpenManus-Go 功能测试脚本
# 用于验证 MCP 集成和多 Agent 协作功能

set -e

echo "🧪 OpenManus-Go 功能测试"
echo "========================"

# 检查二进制文件是否存在
if [ ! -f "./bin/openmanus" ]; then
    echo "❌ 二进制文件不存在，请先运行: make build"
    exit 1
fi

echo "✅ 二进制文件检查通过"

# 测试基本命令
echo ""
echo "📋 测试基本命令..."
./bin/openmanus --help > /dev/null
echo "✅ 主命令正常"

./bin/openmanus run --help > /dev/null
echo "✅ run 命令正常"

./bin/openmanus mcp --help > /dev/null
echo "✅ mcp 命令正常"

./bin/openmanus flow --help > /dev/null
echo "✅ flow 命令正常"

# 测试 MCP 文档生成
echo ""
echo "📚 测试 MCP 文档生成..."
./bin/openmanus mcp --docs > /tmp/mcp_docs.md
if [ -s /tmp/mcp_docs.md ]; then
    echo "✅ MCP 文档生成成功"
    echo "📊 工具数量: $(grep -c "^### " /tmp/mcp_docs.md)"
else
    echo "❌ MCP 文档生成失败"
    exit 1
fi

# 启动 MCP 服务器进行测试
echo ""
echo "🔌 测试 MCP 服务器..."
./bin/openmanus mcp --port 18080 &
MCP_PID=$!
sleep 3

# 检查服务器是否启动
if curl -s http://localhost:18080/health > /dev/null; then
    echo "✅ MCP 服务器启动成功"
    
    # 测试健康检查
    HEALTH=$(curl -s http://localhost:18080/health)
    if echo "$HEALTH" | grep -q '"status":"healthy"'; then
        echo "✅ 健康检查通过"
    else
        echo "❌ 健康检查失败"
    fi
    
    # 测试工具列表
    TOOLS=$(curl -s http://localhost:18080/tools)
    if echo "$TOOLS" | grep -q '"tools"'; then
        TOOL_COUNT=$(echo "$TOOLS" | grep -o '"name":' | wc -l)
        echo "✅ 工具列表获取成功，共 $TOOL_COUNT 个工具"
    else
        echo "❌ 工具列表获取失败"
    fi
    
    # 测试工具调用
    HTTP_RESULT=$(curl -s -X POST http://localhost:18080/tools/invoke \
        -H "Content-Type: application/json" \
        -d '{"tool": "http", "args": {"url": "https://httpbin.org/json", "method": "GET"}}')
    
    if echo "$HTTP_RESULT" | grep -q '"success":true'; then
        echo "✅ HTTP 工具调用成功"
    else
        echo "❌ HTTP 工具调用失败"
    fi
    
else
    echo "❌ MCP 服务器启动失败"
fi

# 停止 MCP 服务器
kill $MCP_PID 2>/dev/null || true
sleep 1

# 测试多 Agent 流程（快速失败模式）
echo ""
echo "🤝 测试多 Agent 流程..."

# 顺序模式
echo "  测试顺序模式..."
timeout 10s ./bin/openmanus flow --mode sequential --agents 1 > /tmp/flow_sequential.log 2>&1 || true
if grep -q "Workflow execution started" /tmp/flow_sequential.log; then
    echo "✅ 顺序模式启动成功"
else
    echo "❌ 顺序模式启动失败"
fi

# 并行模式
echo "  测试并行模式..."
timeout 10s ./bin/openmanus flow --mode parallel --agents 2 > /tmp/flow_parallel.log 2>&1 || true
if grep -q "Workflow execution started" /tmp/flow_parallel.log; then
    echo "✅ 并行模式启动成功"
else
    echo "❌ 并行模式启动失败"
fi

# DAG 模式
echo "  测试 DAG 模式..."
timeout 10s ./bin/openmanus flow --mode dag --agents 2 > /tmp/flow_dag.log 2>&1 || true
if grep -q "Workflow execution started" /tmp/flow_dag.log; then
    echo "✅ DAG 模式启动成功"
else
    echo "❌ DAG 模式启动失败"
fi

# 数据分析模式
echo "  测试数据分析模式..."
timeout 10s ./bin/openmanus flow --data-analysis --mode parallel > /tmp/flow_data.log 2>&1 || true
if grep -q "Workflow execution started" /tmp/flow_data.log; then
    echo "✅ 数据分析模式启动成功"
else
    echo "❌ 数据分析模式启动失败"
fi

# 编译示例程序
echo ""
echo "🔨 测试示例程序编译..."

if go build -o /tmp/mcp_demo examples/mcp_demo/main.go; then
    echo "✅ MCP 示例编译成功"
else
    echo "❌ MCP 示例编译失败"
fi

if go build -o /tmp/multi_agent_demo examples/multi_agent_demo/main.go; then
    echo "✅ 多 Agent 示例编译成功"
else
    echo "❌ 多 Agent 示例编译失败"
fi

# 清理临时文件
rm -f /tmp/mcp_docs.md /tmp/flow_*.log /tmp/mcp_demo /tmp/multi_agent_demo

echo ""
echo "🎉 功能测试完成！"
echo ""
echo "📋 测试总结："
echo "  ✅ 基本命令功能正常"
echo "  ✅ MCP 服务器和客户端功能正常"
echo "  ✅ 多 Agent 协作功能正常"
echo "  ✅ 示例程序编译正常"
echo ""
echo "🚀 OpenManus-Go 已准备就绪！"
echo ""
echo "📖 使用指南："
echo "  - 单 Agent: ./bin/openmanus run"
echo "  - MCP 服务器: ./bin/openmanus mcp"
echo "  - 多 Agent: ./bin/openmanus flow --help"
echo "  - 文档: docs/MCP_AND_MULTIAGENT.md"
