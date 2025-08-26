#!/bin/bash

# 测试 MCP 连接脚本
echo "🔍 测试 MCP 连接..."

# 检查配置文件
if [ ! -f "configs/config.toml" ]; then
    echo "❌ 配置文件不存在: configs/config.toml"
    exit 1
fi

# 提取 MCP 服务器 URL
MCP_URL=$(grep -A 1 "mcp-stock-helper" configs/config.toml | grep "url" | cut -d'"' -f2)

if [ -z "$MCP_URL" ]; then
    echo "❌ 无法从配置文件中提取 MCP URL"
    exit 1
fi

echo "📍 MCP 服务器 URL: $MCP_URL"

# 测试 HTTP 连接
echo "🌐 测试 HTTP 连接..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$MCP_URL/message")

if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "405" ]; then
    echo "✅ HTTP 连接正常 (状态码: $HTTP_STATUS)"
else
    echo "❌ HTTP 连接失败 (状态码: $HTTP_STATUS)"
fi

# 测试 SSE 连接
echo "📡 测试 SSE 连接..."
SSE_URL="$MCP_URL/sse"
echo "📍 SSE URL: $SSE_URL"

# 使用 curl 测试 SSE 连接（5秒超时）
timeout 5 curl -N -H "Accept: text/event-stream" "$SSE_URL" 2>/dev/null | head -10

if [ $? -eq 124 ]; then
    echo "⏰ SSE 连接超时（这是正常的，因为服务器可能没有持续发送数据）"
else
    echo "✅ SSE 连接测试完成"
fi

echo "🎯 测试完成！"
