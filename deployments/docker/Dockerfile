# OpenManus-Go Production Dockerfile
# 构建一个轻量级、安全的生产环境镜像

# 阶段1: 构建阶段
FROM golang:1.21-alpine AS builder

# 设置构建参数
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# 安装构建依赖
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件并下载依赖（利用 Docker 缓存）
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -o openmanus ./cmd/openmanus

# 构建示例程序
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o examples_single_agent ./examples/single_agent
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o examples_mcp_demo ./examples/mcp_demo

# 阶段2: 运行时阶段
FROM alpine:3.18

# 安装运行时依赖
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    chromium \
    nss \
    freetype \
    freetype-dev \
    harfbuzz \
    ttf-freefont \
    curl \
    bash

# 创建用户和组
RUN addgroup -g 1001 -S openmanus && \
    adduser -u 1001 -S openmanus -G openmanus -h /app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制文件
COPY --from=builder /app/openmanus .
COPY --from=builder /app/configs/config.toml ./configs/config.toml
COPY --from=builder /app/internal/prompts ./prompts/

# 创建必要的目录结构
RUN mkdir -p \
    data/traces \
    workspace \
    logs \
    tmp \
    .cache \
    && chown -R openmanus:openmanus /app

# 创建配置目录并设置默认配置
RUN mkdir -p /app/configs && \
    chown -R openmanus:openmanus /app/configs

# 设置环境变量
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/bin/chromium-browser
ENV CHROMIUM_FLAGS="--disable-gpu --disable-dev-shm-usage --disable-setuid-sandbox --no-sandbox"
ENV HOME=/app
ENV PATH=/app:$PATH

# 切换到非 root 用户
USER openmanus

# 暴露端口
EXPOSE 8080

# 添加健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./openmanus tools test || exit 1

# 添加版本信息标签
LABEL maintainer="OpenManus-Go Team" \
      version="${VERSION}" \
      description="OpenManus-Go - 下一代智能 AI Agent 框架" \
      org.opencontainers.image.title="OpenManus-Go" \
      org.opencontainers.image.description="下一代智能 AI Agent 框架，支持多步任务管理和 MCP 集成" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.licenses="MIT"

# 启动命令
ENTRYPOINT ["./openmanus"]
CMD ["run", "--config", "configs/config.toml", "--interactive"]