.PHONY: build test clean run deps

# 构建目标
build:
	go build -o bin/openmanus ./cmd/openmanus

# 运行测试
test:
	go test -v ./...

# 清理构建文件
clean:
	rm -rf bin/

# 运行主程序
run:
	go run ./cmd/openmanus run

# 安装依赖
deps:
	go mod download
	go mod tidy

# 格式化代码
fmt:
	go fmt ./...

# 检查代码
lint:
	golangci-lint run

# 构建 Docker 镜像
docker-build:
	docker build -f deployments/docker/Dockerfile -t openmanus-go:latest .

# 启动开发环境
dev-up:
	docker-compose -f deployments/docker-compose.yaml up -d

# 停止开发环境
dev-down:
	docker-compose -f deployments/docker-compose.yaml down

# 生成 MCP 工具文档
mcp-docs:
	go run ./cmd/openmanus mcp --docs > docs/MCP_TOOLS.md
