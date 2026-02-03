.PHONY: run build tidy test swag

run:
	@go run ./cmd/api

build:
	@go build ./cmd/api

tidy:
	@go mod tidy

test:
	@go test ./...

swag:
	@swag init --dir cmd/api,internal/handler,internal/server,internal/config,internal/model --output docs

