.PHONY: help build test clean run deps fmt lint docker-build docker-push dev-up dev-down mcp-docs examples

# 默认目标
help:
	@echo "OpenManus-Go 构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build                构建主程序"
	@echo "  test                 运行所有测试"
	@echo "  clean                清理构建文件"
	@echo "  run                  运行主程序"
	@echo "  deps                 安装/更新依赖"
	@echo "  fmt                  格式化代码"
	@echo "  lint                 代码检查"
	@echo "  examples             构建所有示例"
	@echo ""
	@echo "Docker 命令:"
	@echo "  docker-build         构建 Docker 镜像"
	@echo "  docker-push          推送 Docker 镜像"
	@echo "  dev-up               启动开发环境"
	@echo "  dev-down             停止开发环境"
	@echo ""
	@echo "工具命令:"
	@echo "  mcp-docs             生成 MCP 工具文档"
	@echo "  test-tools           测试内置工具"
	@echo "  test-mcp             测试 MCP 连接"
	@echo ""
	@echo "示例程序:"
	@echo "  run-single-agent     运行单 Agent 示例"
	@echo "  run-multi-agent      运行多 Agent 示例"
	@echo "  run-mcp-demo         运行 MCP 集成示例"
	@echo "  run-data-analysis    运行数据分析示例"

# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT ?= $(shell git rev-parse HEAD)

# 构建标志
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# 构建主程序
build:
	@echo "🔨 构建 OpenManus-Go..."
	@go build $(LDFLAGS) -o bin/openmanus ./cmd/openmanus
	@echo "✅ 构建完成: bin/openmanus"

# 构建所有程序
build-all: build examples
	@echo "✅ 所有程序构建完成"

# 运行测试
test:
	@echo "🧪 运行测试..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 测试完成，覆盖率报告: coverage.html"

# 运行基准测试
bench:
	@echo "⚡ 运行基准测试..."
	@go test -bench=. -benchmem ./...

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✅ 清理完成"

# 运行主程序
run:
	@echo "🚀 运行 OpenManus-Go..."
	@go run ./cmd/openmanus run --config configs/config.toml

# 交互模式运行
run-interactive:
	@echo "🚀 运行 OpenManus-Go (交互模式)..."
	@go run ./cmd/openmanus run --config configs/config.toml --interactive

# 安装/更新依赖
deps:
	@echo "📦 安装依赖..."
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "✅ 依赖安装完成"

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...
	@goimports -w .
	@echo "✅ 代码格式化完成"

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "警告: golangci-lint 未安装，跳过 lint 检查"; \
		go vet ./...; \
	fi
	@echo "✅ 代码检查完成"

# 构建 Docker 镜像
docker-build:
	@echo "🐳 构建 Docker 镜像..."
	@docker build -f deployments/docker/Dockerfile -t openmanus-go:$(VERSION) .
	@docker tag openmanus-go:$(VERSION) openmanus-go:latest
	@echo "✅ Docker 镜像构建完成: openmanus-go:$(VERSION)"

# 推送 Docker 镜像
docker-push: docker-build
	@echo "📤 推送 Docker 镜像..."
	@docker push openmanus-go:$(VERSION)
	@docker push openmanus-go:latest
	@echo "✅ Docker 镜像推送完成"

# 启动开发环境
dev-up:
	@echo "🚀 启动开发环境..."
	@docker-compose -f deployments/docker-compose.yaml up -d
	@echo "✅ 开发环境已启动"
	@echo "🔗 服务地址:"
	@echo "  - OpenManus-Go: http://localhost:8080"
	@echo "  - Redis: localhost:6379"
	@echo "  - MySQL: localhost:3306"

# 启动完整环境（包括监控）
dev-up-full:
	@echo "🚀 启动完整开发环境..."
	@docker-compose -f deployments/docker-compose.yaml --profile full --profile monitoring up -d
	@echo "✅ 完整开发环境已启动"
	@echo "🔗 服务地址:"
	@echo "  - OpenManus-Go: http://localhost:8080"
	@echo "  - Redis: localhost:6379"
	@echo "  - MySQL: localhost:3306"
	@echo "  - Elasticsearch: http://localhost:9200"
	@echo "  - MinIO: http://localhost:9000"
	@echo "  - Grafana: http://localhost:3000 (admin/admin)"

# 停止开发环境
dev-down:
	@echo "🛑 停止开发环境..."
	@docker-compose -f deployments/docker-compose.yaml down
	@echo "✅ 开发环境已停止"

# 查看环境状态
dev-status:
	@echo "📊 开发环境状态:"
	@docker-compose -f deployments/docker-compose.yaml ps

# 查看日志
dev-logs:
	@echo "📋 查看服务日志:"
	@docker-compose -f deployments/docker-compose.yaml logs -f

# 生成 MCP 工具文档
mcp-docs:
	@echo "📚 生成 MCP 工具文档..."
	@mkdir -p docs
	@go run ./cmd/openmanus tools list > docs/TOOLS_LIST.md
	@echo "✅ MCP 工具文档已生成: docs/TOOLS_LIST.md"

# 测试内置工具
test-tools: build
	@echo "🔧 测试内置工具..."
	@./bin/openmanus tools test
	@echo "✅ 工具测试完成"

# 测试 MCP 连接
test-mcp: build
	@echo "🔌 测试 MCP 连接..."
	@./bin/openmanus run --config configs/config.toml --max-steps 3 "列出可用的 MCP 工具"
	@echo "✅ MCP 连接测试完成"

# 构建示例程序
examples: build-single-agent build-multi-agent build-mcp-demo build-data-analysis

# 构建单 Agent 示例
build-single-agent:
	@echo "🔨 构建单 Agent 示例..."
	@go build -o bin/single_agent ./examples/single_agent
	@echo "✅ 单 Agent 示例构建完成: bin/single_agent"

# 构建多 Agent 示例
build-multi-agent:
	@echo "🔨 构建多 Agent 示例..."
	@go build -o bin/multi_agent_demo ./examples/multi_agent_demo
	@echo "✅ 多 Agent 示例构建完成: bin/multi_agent_demo"

# 构建 MCP 示例
build-mcp-demo:
	@echo "🔨 构建 MCP 示例..."
	@go build -o bin/mcp_demo ./examples/mcp_demo
	@go build -o bin/enhanced_mcp_demo ./examples/enhanced_mcp_demo
	@echo "✅ MCP 示例构建完成"

# 构建数据分析示例
build-data-analysis:
	@echo "🔨 构建数据分析示例..."
	@go build -o bin/data_analysis ./examples/data_analysis
	@echo "✅ 数据分析示例构建完成: bin/data_analysis"

# 运行示例程序
run-single-agent: build-single-agent
	@echo "🚀 运行单 Agent 示例..."
	@./bin/single_agent

run-multi-agent: build-multi-agent
	@echo "🚀 运行多 Agent 示例..."
	@./bin/multi_agent_demo

run-mcp-demo: build-mcp-demo
	@echo "🚀 运行 MCP 基础示例..."
	@./bin/mcp_demo

run-enhanced-mcp-demo: build-mcp-demo
	@echo "🚀 运行增强 MCP 示例..."
	@./bin/enhanced_mcp_demo

run-data-analysis: build-data-analysis
	@echo "🚀 运行数据分析示例..."
	@./bin/data_analysis

# 快速演示
demo: build
	@echo "🎬 运行 OpenManus-Go 演示..."
	@echo "📝 演示任务: 多步任务管理"
	@./bin/openmanus run --config configs/config.toml "创建一个示例文件，内容为当前时间，并保存到 workspace/demo.txt"
	@echo ""
	@echo "📝 演示任务: MCP 集成（如果配置了 MCP 服务器）"
	@./bin/openmanus run --config configs/config.toml "使用 MCP 工具查询天气信息"

# 性能测试
perf-test: build
	@echo "⚡ 性能测试..."
	@echo "测试单步任务性能..."
	@time ./bin/openmanus run --config configs/config.toml "创建一个测试文件"
	@echo ""
	@echo "测试多步任务性能..."
	@time ./bin/openmanus run --config configs/config.toml "创建文件并写入内容，然后读取并显示"

# 安装到系统
install: build
	@echo "📦 安装到系统..."
	@sudo cp bin/openmanus /usr/local/bin/
	@echo "✅ 安装完成: /usr/local/bin/openmanus"

# 从系统卸载
uninstall:
	@echo "🗑️  从系统卸载..."
	@sudo rm -f /usr/local/bin/openmanus
	@echo "✅ 卸载完成"

# 创建发布包
release: clean build-all test
	@echo "📦 创建发布包..."
	@mkdir -p release
	@tar -czf release/openmanus-go-$(VERSION)-linux-amd64.tar.gz bin/ configs/ docs/ README.md LICENSE
	@echo "✅ 发布包已创建: release/openmanus-go-$(VERSION)-linux-amd64.tar.gz"

# 检查代码质量
quality: fmt lint test
	@echo "✅ 代码质量检查完成"

# 开发者快速启动
dev: deps build test-tools
	@echo "🎉 开发环境准备完成！"
	@echo ""
	@echo "快速开始:"
	@echo "  make run              # 运行主程序"
	@echo "  make run-interactive  # 交互模式"
	@echo "  make demo            # 运行演示"
	@echo "  make dev-up          # 启动开发环境"

# 清理所有（包括 Docker）
clean-all: clean dev-down
	@echo "🧹 清理所有文件和容器..."
	@docker system prune -f
	@echo "✅ 全部清理完成"