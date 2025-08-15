# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/openmanus ./cmd/openmanus

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -H -u 10001 app && chown -R app /app
USER app
COPY --from=builder /out/openmanus /app/openmanus
COPY config/ /app/config/
VOLUME ["/data"]
EXPOSE 9000
ENV OPENMANUS_LOG_LEVEL=info
CMD ["/app/openmanus", "serve", "--port", "9000"]
