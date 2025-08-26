#!/bin/bash

# 简单测试 agent 脚本
echo "🧪 测试 agent 基本功能..."

# 设置一个简单的目标
GOAL="请简单介绍一下你自己，用中文回答"

echo "🎯 测试目标: $GOAL"

# 运行 agent（限制步数和时间）
echo "🚀 启动 agent..."

# 在 macOS 上使用 gtimeout，如果没有则直接运行
if command -v gtimeout &> /dev/null; then
    gtimeout 60s ./bin/openmanus run "$GOAL" --max-steps 3 --max-tokens 1000
    EXIT_CODE=$?
elif command -v timeout &> /dev/null; then
    timeout 60s ./bin/openmanus run "$GOAL" --max-steps 3 --max-tokens 1000
    EXIT_CODE=$?
else
    # 没有 timeout 命令，直接运行
    ./bin/openmanus run "$GOAL" --max-steps 3 --max-tokens 1000
    EXIT_CODE=$?
fi

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ 测试成功！"
else
    echo "❌ 测试失败，退出码: $EXIT_CODE"
    exit 1
fi

echo "🎯 测试完成！"
