.PHONY: all

PKG := "github.com/kochabonline/kit"

install: ## Install dependencies
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/favadi/protoc-go-inject-tag@latest
	@go install github.com/google/wire/cmd/wire@latest
	@go install github.com/swaggo/swag/cmd/swag@latest

upgrade: ## Upgrade dependencies
	@go get -u ./...
	@go mod tidy

proto: ## Generate gRPC code
	@protoc -I=. -I=../.. --go_out=. --go_opt=module=${PKG} --go-grpc_out=. --go-grpc_opt=module=${PKG} */*.proto
	@protoc-go-inject-tag -input="*/*.pb.go"
	@go fmt ./...

wire: ## Generate wire code
	@wire ./...

swag: ## Generate swagger docs
	@swag init

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help