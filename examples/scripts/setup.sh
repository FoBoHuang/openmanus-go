#!/bin/bash

# OpenManus-Go 示例环境设置脚本
# 自动设置示例运行所需的环境和依赖

set -e

echo "🚀 OpenManus-Go 示例环境设置"
echo "============================="
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# 检查是否在正确的目录
check_directory() {
    if [[ ! -d "01-quick-start" || ! -d "../configs" ]]; then
        log_error "请在 examples 目录下运行此脚本"
        exit 1
    fi
    log_success "目录检查通过"
}

# 检查 Go 环境
check_go() {
    if ! command -v go &> /dev/null; then
        log_error "未找到 Go 环境，请先安装 Go 1.21+"
        log_info "安装指南: https://golang.org/doc/install"
        exit 1
    fi
    
    GO_VERSION=$(go version | cut -d' ' -f3)
    log_success "Go 环境检查通过 ($GO_VERSION)"
}

# 检查项目是否已构建
check_build() {
    if [[ ! -f "../bin/openmanus" ]]; then
        log_warning "未找到构建的二进制文件，开始构建项目..."
        
        cd ..
        if make build; then
            log_success "项目构建完成"
        else
            log_error "项目构建失败"
            exit 1
        fi
        cd examples
    else
        log_success "项目已构建"
    fi
}

# 创建工作目录
create_directories() {
    log_info "创建必要的目录..."
    
    # 创建 workspace 目录
    mkdir -p ../workspace/{data_processing/{input,output,temp},traces,logs}
    
    # 创建示例数据目录
    mkdir -p ../workspace/examples_data
    
    log_success "目录创建完成"
}

# 创建示例配置文件
setup_config() {
    log_info "设置配置文件..."
    
    CONFIG_PATH="../configs/config.toml"
    EXAMPLE_CONFIG_PATH="../configs/config.example.toml"
    
    if [[ ! -f "$CONFIG_PATH" ]]; then
        if [[ -f "$EXAMPLE_CONFIG_PATH" ]]; then
            cp "$EXAMPLE_CONFIG_PATH" "$CONFIG_PATH"
            log_success "配置文件创建完成"
            log_warning "请编辑 $CONFIG_PATH 设置你的 LLM API Key"
        else
            log_error "未找到配置模板文件"
            exit 1
        fi
    else
        log_success "配置文件已存在"
    fi
}

# 创建示例数据文件
create_sample_data() {
    log_info "创建示例数据文件..."
    
    # 创建示例 CSV 数据
    cat > ../workspace/examples_data/sample_sales.csv << 'EOF'
Date,Product,Quantity,Amount
2024-01-01,Product A,100,1000.00
2024-01-01,Product B,80,1600.00
2024-01-02,Product A,120,1200.00
2024-01-02,Product C,60,900.00
2024-01-03,Product B,90,1800.00
2024-01-03,Product C,70,1050.00
EOF

    # 创建示例 JSON 数据
    cat > ../workspace/examples_data/sample_config.json << 'EOF'
{
    "app_name": "OpenManus-Go Demo",
    "version": "1.0.0",
    "author": "OpenManus Team",
    "features": [
        "Agent Framework",
        "Tool System",
        "MCP Integration"
    ],
    "settings": {
        "max_retries": 3,
        "timeout": 30,
        "debug": false
    }
}
EOF

    log_success "示例数据文件创建完成"
}

# 验证依赖服务
check_optional_services() {
    log_info "检查可选服务..."
    
    # 检查 Redis
    if command -v redis-cli &> /dev/null; then
        if redis-cli ping &> /dev/null; then
            log_success "Redis 服务可用"
        else
            log_warning "Redis 已安装但未运行"
        fi
    else
        log_warning "Redis 未安装 (可选，用于缓存示例)"
    fi
    
    # 检查 Chrome/Chromium (用于浏览器示例)
    if command -v google-chrome &> /dev/null || command -v chromium &> /dev/null; then
        log_success "Chrome/Chromium 可用 (浏览器示例)"
    else
        log_warning "Chrome/Chromium 未找到 (浏览器示例将无法运行)"
    fi
    
    # 检查 Docker
    if command -v docker &> /dev/null; then
        log_success "Docker 可用 (容器示例)"
    else
        log_warning "Docker 未安装 (容器示例将无法运行)"
    fi
}

# 运行基础测试
run_basic_tests() {
    log_info "运行基础测试..."
    
    # 测试配置验证
    if ../bin/openmanus config validate --config ../configs/config.toml &> /dev/null; then
        log_success "配置验证通过"
    else
        log_warning "配置验证失败，请检查 API Key 设置"
    fi
    
    # 测试工具列表
    if ../bin/openmanus tools list --config ../configs/config.toml &> /dev/null; then
        log_success "工具系统正常"
    else
        log_warning "工具系统测试失败"
    fi
}

# 显示运行指南
show_usage_guide() {
    echo
    echo "🎉 环境设置完成！"
    echo
    echo "📚 快速开始指南:"
    echo "================"
    echo
    echo "1. 设置 API Key (重要):"
    echo "   编辑 ../configs/config.toml"
    echo "   设置 [llm] 部分的 api_key"
    echo
    echo "2. 运行 Hello World 示例:"
    echo "   cd 01-quick-start/hello-world"
    echo "   go run main.go"
    echo
    echo "3. 使用 CLI 工具:"
    echo "   ../bin/openmanus run \"创建一个测试文件\""
    echo
    echo "4. 运行所有示例:"
    echo "   ./scripts/run-all.sh"
    echo
    echo "5. 测试示例:"
    echo "   ./scripts/test-examples.sh"
    echo
    echo "📁 重要目录:"
    echo "============"
    echo "  ../workspace/          - 工作目录 (文件操作)"
    echo "  ../workspace/traces/   - 执行轨迹"
    echo "  ../configs/config.toml - 配置文件"
    echo "  ../bin/openmanus      - CLI 工具"
    echo
    echo "🔧 故障排除:"
    echo "============"
    echo "  - 如果示例运行失败，检查 API Key 设置"
    echo "  - 如果工具调用失败，检查目录权限"
    echo "  - 如果网络请求失败，检查网络连接"
    echo
    echo "💡 提示:"
    echo "========"
    echo "  - 每个示例目录都有详细的 README.md"
    echo "  - 可以修改任务描述来测试不同场景"
    echo "  - 查看 workspace 目录验证文件操作结果"
    echo
}

# 主执行流程
main() {
    echo "开始设置 OpenManus-Go 示例环境..."
    echo
    
    check_directory
    check_go
    check_build
    create_directories
    setup_config
    create_sample_data
    check_optional_services
    run_basic_tests
    show_usage_guide
}

# 执行主函数
main "$@"
