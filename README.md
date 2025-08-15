# openmanus-go (Phase 2)

CLI + Tools + Agent + Flow + HTTP server

## Quickstart
```bash
go mod tidy
go build -o bin/openmanus ./cmd/openmanus
./bin/openmanus tools
./bin/openmanus run --prompt "echo: hello tools"
./bin/openmanus run --prompt "fetch http https://example.com"
```
