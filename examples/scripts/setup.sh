#!/bin/bash

# OpenManus-Go 示例程序环境设置脚本
# 用于初始化运行示例程序所需的环境

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

# 检查 Go 环境
check_go_environment() {
    log_info "检查 Go 环境..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.21+"
        echo "安装指南: https://golang.org/doc/install"
        exit 1
    fi
    
    go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | head -1)
    log_success "Go 版本: $go_version"
    
    # 检查 Go 版本（简单检查）
    if [[ "$go_version" < "go1.21" ]]; then
        log_warning "建议使用 Go 1.21+ 版本"
    fi
}

# 检查项目结构
check_project_structure() {
    log_info "检查项目结构..."
    
    if [ ! -f "../go.mod" ]; then
        log_error "请在 examples 目录下运行此脚本"
        exit 1
    fi
    
    if [ ! -f "../Makefile" ]; then
        log_warning "Makefile 不存在，将使用 go build 命令"
    fi
    
    log_success "项目结构检查通过"
}

# 创建必要目录
create_directories() {
    log_info "创建必要目录..."
    
    directories=("../workspace" "../workspace/traces" "../data" "../logs")
    
    for dir in "${directories[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            log_success "创建目录: $dir"
        else
            log_info "目录已存在: $dir"
        fi
    done
}

# 设置配置文件
setup_config() {
    log_info "设置配置文件..."
    
    config_dir="../configs"
    config_file="$config_dir/config.toml"
    example_config="$config_dir/config.example.toml"
    
    if [ ! -f "$config_file" ]; then
        if [ -f "$example_config" ]; then
            cp "$example_config" "$config_file"
            log_success "已复制配置模板到 $config_file"
            log_warning "请编辑 $config_file 设置您的 API Key"
        else
            log_error "配置模板文件不存在: $example_config"
        fi
    else
        log_info "配置文件已存在: $config_file"
    fi
}

# 下载依赖
download_dependencies() {
    log_info "下载项目依赖..."
    
    cd ..
    if go mod download && go mod tidy; then
        log_success "依赖下载完成"
    else
        log_error "依赖下载失败"
        exit 1
    fi
    cd examples
}

# 构建项目
build_project() {
    log_info "构建项目..."
    
    cd ..
    if [ -f "Makefile" ]; then
        if make build > /dev/null 2>&1; then
            log_success "项目构建成功 (使用 Makefile)"
        else
            log_error "项目构建失败"
            exit 1
        fi
    else
        if go build -o bin/openmanus ./cmd/openmanus; then
            log_success "项目构建成功 (使用 go build)"
        else
            log_error "项目构建失败"
            exit 1
        fi
    fi
    cd examples
}

# 检查可选依赖
check_optional_dependencies() {
    log_info "检查可选依赖..."
    
    # 检查 Chrome/Chromium（用于浏览器工具）
    if command -v google-chrome &> /dev/null || command -v chromium-browser &> /dev/null || command -v chromium &> /dev/null; then
        log_success "浏览器工具依赖: Chrome/Chromium 已安装"
    else
        log_warning "浏览器工具依赖: Chrome/Chromium 未安装，浏览器工具将不可用"
        echo "  安装命令 (Ubuntu): sudo apt-get install chromium-browser"
        echo "  安装命令 (macOS): brew install chromium"
    fi
    
    # 检查 Redis
    if command -v redis-cli &> /dev/null; then
        if redis-cli ping &> /dev/null; then
            log_success "Redis 工具依赖: Redis 服务正在运行"
        else
            log_warning "Redis 工具依赖: Redis 已安装但未运行"
            echo "  启动命令: redis-server"
        fi
    else
        log_warning "Redis 工具依赖: Redis 未安装，Redis 工具将不可用"
        echo "  安装命令 (Ubuntu): sudo apt-get install redis-server"
        echo "  安装命令 (macOS): brew install redis"
    fi
    
    # 检查 MySQL
    if command -v mysql &> /dev/null; then
        log_success "MySQL 工具依赖: MySQL 客户端已安装"
    else
        log_warning "MySQL 工具依赖: MySQL 客户端未安装，MySQL 工具将不可用"
        echo "  安装命令 (Ubuntu): sudo apt-get install mysql-client"
        echo "  安装命令 (macOS): brew install mysql"
    fi
}

# 运行基本测试
run_basic_tests() {
    log_info "运行基本测试..."
    
    if [ -x "scripts/test-examples.sh" ]; then
        if ./scripts/test-examples.sh > /dev/null 2>&1; then
            log_success "基本测试通过"
        else
            log_warning "基本测试失败，请检查代码"
        fi
    else
        log_warning "测试脚本不存在或无执行权限"
    fi
}

# 显示使用指南
show_usage_guide() {
    echo
    echo "🎉 环境设置完成！"
    echo "================"
    echo
    echo "📚 下一步操作："
    echo
    echo "1. 设置 API Key（重要！）"
    echo "   编辑文件: ../configs/config.toml"
    echo "   设置 api_key = \"your-actual-api-key\""
    echo
    echo "2. 运行示例程序"
    echo "   测试所有示例: ./scripts/test-examples.sh"
    echo "   运行所有示例: ./scripts/run-all.sh"
    echo "   运行特定示例: ./scripts/run-all.sh --examples basic"
    echo
    echo "3. 手动运行示例"
    echo "   cd basic/01-hello-world && go run main.go"
    echo "   cd basic/02-tool-usage && go run main.go"
    echo "   cd basic/03-configuration && go run main.go"
    echo
    echo "4. MCP 服务器测试"
    echo "   启动服务器: cd mcp/01-mcp-server && go run main.go"
    echo "   测试客户端: cd mcp/02-mcp-client && go run main.go"
    echo
    echo "5. 查看文档"
    echo "   主文档: cat README.md"
    echo "   示例文档: cat basic/01-hello-world/README.md"
    echo
    echo "💡 提示："
    echo "  - 所有示例都有详细的 README.md 文档"
    echo "  - 没有 API Key 时示例会进入演示模式"
    echo "  - 使用 --help 查看脚本帮助信息"
    echo
}

# 主函数
main() {
    echo "🔧 OpenManus-Go 示例程序环境设置"
    echo "================================"
    echo
    
    # 解析命令行参数
    SKIP_BUILD=false
    SKIP_DEPS=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-deps)
                SKIP_DEPS=true
                shift
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --skip-build     跳过项目构建"
                echo "  --skip-deps      跳过依赖检查"
                echo "  --help          显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 执行设置步骤
    check_go_environment
    check_project_structure
    create_directories
    setup_config
    
    if [ "$SKIP_DEPS" = false ]; then
        download_dependencies
    fi
    
    if [ "$SKIP_BUILD" = false ]; then
        build_project
    fi
    
    check_optional_dependencies
    run_basic_tests
    show_usage_guide
}

# 运行主函数
main "$@"
