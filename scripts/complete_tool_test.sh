#!/bin/bash

echo "=== 第1步：调用 DeepSeek API 获取工具调用指令 ==="
response=$(curl -s -X POST "https://api.deepseek.com/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-a83be43ee33544f186e5fd7b5dab2ca8" \
  -d '{
    "model": "deepseek-chat",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant that can use tools to accomplish tasks."
      },
      {
        "role": "user", 
        "content": "请使用fs工具在workspace目录中创建一个hello.txt文件，内容为Hello World"
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "fs",
          "description": "文件系统操作工具，支持读取、写入、列表、删除等操作",
          "parameters": {
            "type": "object",
            "properties": {
              "operation": {
                "type": "string",
                "description": "操作类型：read, write, list, delete, mkdir, exists, stat"
              },
              "path": {
                "type": "string", 
                "description": "文件或目录路径"
              },
              "content": {
                "type": "string",
                "description": "写入内容（仅用于 write 操作）"
              },
              "recursive": {
                "type": "boolean",
                "description": "是否递归操作（用于 list, mkdir 等）"
              }
            },
            "required": ["operation", "path"]
          }
        }
      }
    ],
    "tool_choice": "auto",
    "temperature": 0.1
  }')

echo "API 响应："
echo "$response" | jq .

echo -e "\n=== 第2步：解析工具调用参数 ==="
tool_call=$(echo "$response" | jq -r '.choices[0].message.tool_calls[0].function.arguments')
echo "工具调用参数：$tool_call"

# 解析参数
operation=$(echo "$tool_call" | jq -r '.operation')
path=$(echo "$tool_call" | jq -r '.path')
content=$(echo "$tool_call" | jq -r '.content')

echo "操作类型：$operation"
echo "文件路径：$path"
echo "文件内容：$content"

echo -e "\n=== 第3步：手动执行工具操作 ==="
if [ "$operation" = "write" ]; then
    # 确保目录存在
    mkdir -p "$(dirname "$path")"
    # 写入文件
    echo "$content" > "$path"
    echo "✅ 文件已创建：$path"
    echo "文件内容："
    cat "$path"
else
    echo "❌ 不支持的操作：$operation"
fi

echo -e "\n=== 第4步：验证结果 ==="
if [ -f "$path" ]; then
    echo "✅ 文件存在"
    echo "文件大小：$(wc -c < "$path") 字节"
    echo "文件内容：$(cat "$path")"
else
    echo "❌ 文件不存在"
fi
