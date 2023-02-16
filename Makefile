folder: ## create folder structure
	@mkdir -p db
	@mkdir -p log
	@mkdir -p opentracing
	@mkdir -p redis
	@mkdir -p registry

dep: ## Install required packages
	@GO111MODULE=on go get -v github.com/golangci/golangci-lint/cmd/golangci-lint

mod: ## Install go library
	@go mod tidy
	@go mod vendor

lint: ## Run linter
	@golangci-lint run ./...
