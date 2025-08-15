BINARY?=openmanus

.PHONY: build run docker lint
build:
	go mod tidy
	go build -o bin/$(BINARY) ./cmd/openmanus

run:
	./bin/$(BINARY) serve --port 9000

docker:
	docker build -t openmanus-go:latest .

lint:
	golangci-lint run
