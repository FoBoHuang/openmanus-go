#!/bin/bash

# DeepSeek 工具调用测试
curl -X POST "https://api.deepseek.com/v1/chat/completions" \
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
  }'
