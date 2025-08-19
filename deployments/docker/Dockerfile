# OpenManus-Go Dockerfile
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o openmanus ./cmd/openmanus

# 最终阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata chromium

# 创建用户
RUN addgroup -g 1001 -S openmanus && \
    adduser -u 1001 -S openmanus -G openmanus

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/openmanus .

# 复制配置文件
COPY --from=builder /app/configs/config.example.toml ./config.toml
COPY --from=builder /app/internal/prompts ./prompts/

# 创建数据目录
RUN mkdir -p data/traces workspace && \
    chown -R openmanus:openmanus /app

# 切换到非 root 用户
USER openmanus

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/bin/chromium-browser

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./openmanus tools test || exit 1

# 启动命令
ENTRYPOINT ["./openmanus"]
CMD ["run", "--interactive"]
