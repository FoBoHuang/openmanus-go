#!/bin/bash

# OpenManus-Go 示例运行脚本
# 自动运行所有示例，展示框架的完整功能

set -e

echo "🚀 OpenManus-Go 示例演示"
echo "========================"
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_step() {
    echo -e "${PURPLE}📋 $1${NC}"
}

log_result() {
    echo -e "${CYAN}📊 $1${NC}"
}

# 检查环境
check_environment() {
    log_info "检查运行环境..."
    
    # 检查目录
    if [[ ! -d "01-quick-start" ]]; then
        log_error "请在 examples 目录下运行此脚本"
        exit 1
    fi
    
    # 检查配置文件
    if [[ ! -f "../configs/config.toml" ]]; then
        log_warning "配置文件不存在，将使用默认配置"
        log_info "建议运行 ./scripts/setup.sh 先设置环境"
    fi
    
    # 检查二进制文件
    if [[ ! -f "../bin/openmanus" ]]; then
        log_warning "未找到构建的二进制文件"
        log_info "将只运行 Go 源码示例"
    fi
    
    log_success "环境检查完成"
}

# 运行单个示例
run_example() {
    local category="$1"
    local name="$2"
    local description="$3"
    local path="$4"
    
    echo
    log_step "运行示例: $category/$name"
    echo "📝 描述: $description"
    echo "📁 路径: $path"
    echo
    
    if [[ ! -d "$path" ]]; then
        log_error "示例目录不存在: $path"
        return 1
    fi
    
    if [[ ! -f "$path/main.go" ]]; then
        log_error "示例文件不存在: $path/main.go"
        return 1
    fi
    
    echo "🔄 执行中..."
    echo "----------------------------------------"
    
    # 进入示例目录并运行
    cd "$path"
    
    # 设置超时时间（避免示例运行过长时间）
    if timeout 300s go run main.go; then
        log_success "示例运行成功"
        local result=0
    else
        log_warning "示例运行超时或失败"
        local result=1
    fi
    
    cd - > /dev/null
    echo "----------------------------------------"
    
    return $result
}

# 运行 CLI 示例
run_cli_examples() {
    if [[ ! -f "../bin/openmanus" ]]; then
        log_warning "跳过 CLI 示例（二进制文件不存在）"
        return
    fi
    
    echo
    log_step "运行 CLI 示例"
    echo
    
    local cli_examples=(
        "创建一个 hello_cli.txt 文件，内容为当前时间"
        "检查 workspace 目录下的文件数量"
        "获取 https://httpbin.org/uuid 的内容"
    )
    
    for example in "${cli_examples[@]}"; do
        echo "🔄 执行 CLI 任务: $example"
        echo "命令: ../bin/openmanus run \"$example\""
        echo
        
        if timeout 60s ../bin/openmanus run --config ../configs/config.toml "$example"; then
            log_success "CLI 任务完成"
        else
            log_warning "CLI 任务失败或超时"
        fi
        echo
    done
}

# 展示结果统计
show_statistics() {
    local total=$1
    local success=$2
    local failed=$((total - success))
    
    echo
    echo "📊 运行统计"
    echo "============"
    log_result "总示例数: $total"
    log_result "成功运行: $success"
    log_result "运行失败: $failed"
    
    if [[ $total -gt 0 ]]; then
        local success_rate=$(( success * 100 / total ))
        log_result "成功率: ${success_rate}%"
    fi
    
    echo
}

# 展示生成的文件
show_generated_files() {
    echo "📁 查看生成的文件"
    echo "=================="
    
    if [[ -d "../workspace" ]]; then
        echo "工作目录内容:"
        find ../workspace -type f -name "*.txt" -o -name "*.json" -o -name "*.csv" | head -10 | while read file; do
            echo "  📄 $file"
        done
        
        local file_count=$(find ../workspace -type f | wc -l)
        if [[ $file_count -gt 10 ]]; then
            echo "  ... 还有 $((file_count - 10)) 个文件"
        fi
    else
        echo "  📁 workspace 目录不存在"
    fi
    
    echo
}

# 主要示例列表
declare -a EXAMPLES=(
    # 格式: "类别|名称|描述|路径"
    "01-快速入门|Hello World|最基础的框架使用示例|01-quick-start/hello-world"
    "01-快速入门|基础任务|展示各种基础任务执行|01-quick-start/basic-tasks"
    "02-工具使用|文件系统|文件系统工具完整演示|02-tool-usage/filesystem"
    "03-MCP集成|MCP客户端|MCP协议集成和外部服务调用|03-mcp-integration/mcp-client"
    "04-实际应用|数据处理|真实数据处理工作流演示|04-real-world/data-processing"
)

# 主执行函数
main() {
    echo "开始运行 OpenManus-Go 示例演示..."
    echo
    
    check_environment
    
    local total_examples=0
    local successful_examples=0
    
    # 显示即将运行的示例
    echo "📋 将要运行的示例:"
    echo "=================="
    for example in "${EXAMPLES[@]}"; do
        IFS='|' read -r category name description path <<< "$example"
        echo "  🔸 $category - $name: $description"
        ((total_examples++))
    done
    echo
    
    # 询问用户是否继续
    read -p "是否继续运行所有示例? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "用户取消运行"
        exit 0
    fi
    
    echo
    log_info "开始运行示例..."
    
    # 运行所有示例
    for example in "${EXAMPLES[@]}"; do
        IFS='|' read -r category name description path <<< "$example"
        
        if run_example "$category" "$name" "$description" "$path"; then
            ((successful_examples++))
        fi
        
        # 示例间暂停
        if [[ ${#EXAMPLES[@]} -gt 1 ]]; then
            echo
            echo "按 Enter 继续下一个示例，或 Ctrl+C 退出..."
            read -r
        fi
    done
    
    # 运行 CLI 示例
    run_cli_examples
    
    # 显示统计信息
    show_statistics $total_examples $successful_examples
    
    # 显示生成的文件
    show_generated_files
    
    # 最终提示
    echo "🎉 示例演示完成！"
    echo
    echo "📚 下一步建议:"
    echo "=============="
    echo "  1. 查看 workspace 目录中生成的文件"
    echo "  2. 编辑配置文件 ../configs/config.toml 设置 API Key"
    echo "  3. 重新运行示例体验完整功能"
    echo "  4. 阅读各示例目录中的 README.md"
    echo "  5. 尝试修改任务描述测试不同场景"
    echo
    echo "💡 提示:"
    echo "========"
    echo "  - 运行 ./scripts/test-examples.sh 进行自动化测试"
    echo "  - 运行 ./scripts/setup.sh 重新设置环境"
    echo "  - 查看 ../README.md 了解更多功能"
}

# 信号处理
trap 'echo; log_warning "用户中断运行"; exit 130' INT

# 执行主函数
main "$@"