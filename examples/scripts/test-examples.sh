#!/bin/bash

# OpenManus-Go 示例程序测试脚本
# 用于测试示例程序的编译和基本功能（不需要 LLM API）

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 测试示例编译
test_compilation() {
    local example_path=$1
    local example_name=$2
    
    if [ ! -d "$example_path" ]; then
        log_warning "示例不存在: $example_path"
        return 1
    fi
    
    log_info "测试编译: $example_name"
    
    cd "$example_path"
    
    if go build -o /tmp/example_test main.go 2>/dev/null; then
        log_success "$example_name 编译成功"
        rm -f /tmp/example_test
        cd - > /dev/null
        return 0
    else
        log_error "$example_name 编译失败"
        cd - > /dev/null
        return 1
    fi
}

# 主函数
main() {
    echo "🧪 OpenManus-Go 示例程序测试器"
    echo "==============================="
    echo
    
    # 检查项目根目录
    if [ ! -f "../go.mod" ]; then
        log_error "请在 examples 目录下运行此脚本"
        exit 1
    fi
    
    # 构建项目
    log_info "构建项目..."
    cd ..
    if make build > /dev/null 2>&1; then
        log_success "项目构建成功"
    else
        log_error "项目构建失败"
        exit 1
    fi
    cd examples
    
    # 测试统计
    total_tests=0
    passed_tests=0
    
    echo "🔍 开始测试示例程序..."
    echo
    
    # 定义示例列表
    examples=(
        "basic/01-hello-world:Hello World 示例"
        "basic/02-tool-usage:工具使用示例"
        "basic/03-configuration:配置管理示例"
        "mcp/01-mcp-server:MCP 服务器示例"
        "mcp/02-mcp-client:MCP 客户端示例"
    )
    
    # 测试每个示例
    for example_entry in "${examples[@]}"; do
        IFS=':' read -r example_path example_name <<< "$example_entry"
        
        total_tests=$((total_tests + 1))
        
        echo "测试: $example_name"
        echo "--------------------"
        
        # 编译测试
        if test_compilation "$example_path" "$example_name"; then
            passed_tests=$((passed_tests + 1))
        fi
        
        echo
    done
    
    # 额外测试
    echo "🔧 运行额外测试..."
    echo
    
    # 测试脚本权限
    if [ -x "scripts/run-all.sh" ]; then
        log_success "运行脚本权限正确"
    else
        log_error "运行脚本缺少执行权限"
    fi
    
    # 测试 README 文件
    if [ -f "README.md" ]; then
        log_success "README.md 文件存在"
        if grep -q "OpenManus-Go" README.md; then
            log_success "README.md 内容正确"
        else
            log_warning "README.md 内容可能不完整"
        fi
    else
        log_error "README.md 文件缺失"
    fi
    
    # 测试目录结构
    expected_dirs=("basic" "mcp" "scripts")
    for dir in "${expected_dirs[@]}"; do
        if [ -d "$dir" ]; then
            log_success "目录结构正确: $dir/"
        else
            log_error "目录缺失: $dir/"
        fi
    done
    
    # 输出测试总结
    echo
    echo "📊 测试总结"
    echo "=========="
    echo "总测试数: $total_tests"
    echo "通过测试: $passed_tests"
    echo "失败测试: $((total_tests - passed_tests))"
    
    echo
    if [ $passed_tests -eq $total_tests ]; then
        log_success "🎉 所有测试通过！"
        echo
        echo "💡 下一步："
        echo "  1. 运行 './scripts/run-all.sh' 执行所有示例"
        echo "  2. 设置 API Key 体验完整功能"
        echo "  3. 查看各示例的 README.md 了解详细用法"
        exit 0
    else
        log_error "❌ 部分测试失败，请检查代码"
        exit 1
    fi
}

# 运行主函数
main "$@"