# 快速使用

## 解压并设置环境变量

```bash
export OPENAI_API_KEY=sk-xxxx
```

---

## 本地运行（CLI）

### 列工具
```bash
go run ./cmd/openmanus tools
```

### 只规划（让 AI 看工具 → 产出 JSON steps）
```bash
echo '抓取 https://example.com 然后用正则提取所有 http 链接' | go run ./cmd/openmanus plan --stdin
```

### 计划 + 执行（完全自治）
```bash
echo '访问 https://example.com 然后用正则提取所有 http 链接' | go run ./cmd/openmanus run --stdin
```

---

## HTTP 服务

启动：
```bash
go run ./cmd/openmanus serve --port 9000
```

### POST `/v1/flow/run`

#### - 走 planner（自治）
```bash
curl -X POST http://localhost:9000/v1/flow/run   -d '{"prompt":"访问 https://example.com 并提取所有 http 链接","mode":"plan"}'   -H 'Content-Type: application/json'
```

#### - 直接给 steps（手工编排）
```bash
curl -X POST http://localhost:9000/v1/flow/run   -d '{
    "mode": "steps",
    "steps": [
      {
        "kind": "tool",
        "name": "http_get",
        "input": { "url": "https://example.com" }
      },
      {
        "kind": "tool",
        "name": "regex_extract",
        "input": {
          "text": "{{prev.body}}",
          "pattern": "https?://[^\\s\\\"]+"
        }
      }
    ]
  }'   -H 'Content-Type: application/json'
```

---

## Docker & K8s

### 本地镜像
```bash
docker build -t openmanus-go:latest .
```

### K8s 部署（示例清单在 `deploy/k8s/`）
```bash
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/pvc.yaml
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
```

---

## 监控与健康检查
- Prometheus 指标：`/metrics`
- 健康检查：`/healthz`

---

## GitHub Actions
`.github/workflows/ci.yml` 已包含：
- **build**
- **lint**（`golangci-lint`）
- **docker**（build + push GHCR）

> 将 GHCR 推送目的地改成你的 `org/repo` 即可。
