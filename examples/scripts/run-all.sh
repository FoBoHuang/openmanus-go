#!/bin/bash

# OpenManus-Go 示例程序运行脚本
# 用于批量运行所有示例程序

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

# 检查项目根目录
check_project_root() {
    if [ ! -f "../go.mod" ]; then
        log_error "请在 examples 目录下运行此脚本"
        exit 1
    fi
    
    if [ ! -f "../configs/config.toml" ]; then
        log_warning "配置文件不存在，某些示例可能无法完整运行"
        log_info "建议运行: cp ../configs/config.example.toml ../configs/config.toml"
    fi
}

# 构建项目
build_project() {
    log_info "构建项目..."
    cd ..
    if make build > /dev/null 2>&1; then
        log_success "项目构建成功"
    else
        log_error "项目构建失败"
        exit 1
    fi
    cd examples
}

# 运行单个示例
run_example() {
    local example_path=$1
    local example_name=$2
    local timeout=${3:-30}
    
    if [ ! -d "$example_path" ]; then
        log_warning "示例不存在: $example_path"
        return 1
    fi
    
    log_info "运行示例: $example_name"
    echo "----------------------------------------"
    
    cd "$example_path"
    
    # 设置超时运行
    timeout "${timeout}s" go run main.go 2>&1 || {
        local exit_code=$?
        if [ $exit_code -eq 124 ]; then
            log_warning "$example_name 运行超时 (${timeout}s)"
        else
            log_error "$example_name 运行失败 (退出码: $exit_code)"
        fi
        cd - > /dev/null
        return $exit_code
    }
    
    cd - > /dev/null
    log_success "$example_name 运行完成"
    echo
}

# 检查 MCP 服务器
check_mcp_server() {
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# 启动 MCP 服务器
start_mcp_server() {
    log_info "启动 MCP 服务器..."
    cd mcp/01-mcp-server
    go run main.go > /dev/null 2>&1 &
    local server_pid=$!
    cd - > /dev/null
    
    # 等待服务器启动
    for i in {1..10}; do
        if check_mcp_server; then
            log_success "MCP 服务器已启动 (PID: $server_pid)"
            echo $server_pid > /tmp/mcp_server.pid
            return 0
        fi
        sleep 1
    done
    
    log_error "MCP 服务器启动失败"
    kill $server_pid 2>/dev/null || true
    return 1
}

# 停止 MCP 服务器
stop_mcp_server() {
    if [ -f /tmp/mcp_server.pid ]; then
        local server_pid=$(cat /tmp/mcp_server.pid)
        if kill $server_pid 2>/dev/null; then
            log_success "MCP 服务器已停止"
        fi
        rm -f /tmp/mcp_server.pid
    fi
}

# 主函数
main() {
    echo "🚀 OpenManus-Go 示例程序批量运行器"
    echo "=================================="
    echo
    
    # 解析命令行参数
    SKIP_BUILD=false
    EXAMPLES_TO_RUN=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --examples)
                EXAMPLES_TO_RUN="$2"
                shift 2
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --skip-build     跳过项目构建"
                echo "  --examples LIST  只运行指定的示例 (逗号分隔)"
                echo "  --help          显示帮助信息"
                echo
                echo "示例:"
                echo "  $0                           # 运行所有示例"
                echo "  $0 --skip-build              # 跳过构建，运行所有示例"
                echo "  $0 --examples basic,mcp      # 只运行基础和 MCP 示例"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 检查环境
    check_project_root
    
    # 构建项目
    if [ "$SKIP_BUILD" = false ]; then
        build_project
    fi
    
    # 定义示例列表
    declare -A examples
    examples["basic-hello"]="basic/01-hello-world,Hello World 示例,10"
    examples["basic-tools"]="basic/02-tool-usage,工具使用示例,20"
    examples["basic-config"]="basic/03-configuration,配置管理示例,10"
    examples["mcp-server"]="mcp/01-mcp-server,MCP 服务器示例,5"
    examples["mcp-client"]="mcp/02-mcp-client,MCP 客户端示例,15"
    
    # 过滤要运行的示例
    if [ -n "$EXAMPLES_TO_RUN" ]; then
        IFS=',' read -ra EXAMPLE_LIST <<< "$EXAMPLES_TO_RUN"
        filtered_examples=""
        for example in "${EXAMPLE_LIST[@]}"; do
            case $example in
                basic)
                    filtered_examples="$filtered_examples basic-hello basic-tools basic-config"
                    ;;
                mcp)
                    filtered_examples="$filtered_examples mcp-server mcp-client"
                    ;;
                *)
                    if [[ -v examples[$example] ]]; then
                        filtered_examples="$filtered_examples $example"
                    else
                        log_warning "未知示例: $example"
                    fi
                    ;;
            esac
        done
        EXAMPLES_TO_RUN="$filtered_examples"
    else
        EXAMPLES_TO_RUN="${!examples[@]}"
    fi
    
    # 统计信息
    total_examples=0
    success_count=0
    failed_examples=""
    
    # 运行示例
    for example_key in $EXAMPLES_TO_RUN; do
        if [[ -v examples[$example_key] ]]; then
            IFS=',' read -ra example_info <<< "${examples[$example_key]}"
            example_path="${example_info[0]}"
            example_name="${example_info[1]}"
            timeout="${example_info[2]}"
            
            total_examples=$((total_examples + 1))
            
            # 特殊处理 MCP 示例
            if [[ $example_key == "mcp-server" ]]; then
                # MCP 服务器示例特殊处理（后台运行）
                log_info "启动 MCP 服务器示例（后台运行）..."
                if start_mcp_server; then
                    success_count=$((success_count + 1))
                    sleep 2  # 给服务器一些启动时间
                else
                    failed_examples="$failed_examples $example_name"
                fi
            elif [[ $example_key == "mcp-client" ]]; then
                # MCP 客户端需要服务器运行
                if ! check_mcp_server; then
                    log_warning "MCP 服务器未运行，跳过客户端示例"
                    continue
                fi
                if run_example "$example_path" "$example_name" "$timeout"; then
                    success_count=$((success_count + 1))
                else
                    failed_examples="$failed_examples $example_name"
                fi
            else
                # 普通示例
                if run_example "$example_path" "$example_name" "$timeout"; then
                    success_count=$((success_count + 1))
                else
                    failed_examples="$failed_examples $example_name"
                fi
            fi
        fi
    done
    
    # 清理
    stop_mcp_server
    
    # 输出总结
    echo "📊 运行总结"
    echo "=========="
    echo "总示例数: $total_examples"
    echo "成功运行: $success_count"
    echo "运行失败: $((total_examples - success_count))"
    
    if [ -n "$failed_examples" ]; then
        echo "失败示例:$failed_examples"
    fi
    
    echo
    if [ $success_count -eq $total_examples ]; then
        log_success "🎉 所有示例运行成功！"
        exit 0
    else
        log_warning "⚠️  部分示例运行失败"
        exit 1
    fi
}

# 信号处理
trap 'stop_mcp_server; exit 130' INT TERM

# 运行主函数
main "$@"
