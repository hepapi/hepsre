.PHONY: help build run test clean docker-build install-deps

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

kubectl-plugin: build ## Build and install kubectl plugin
	@echo "Installing kubectl plugin..."
	@mkdir -p $(HOME)/.local/bin
	@cp bin/hepsre $(HOME)/.local/bin/kubectl-hepsre
	@chmod +x $(HOME)/.local/bin/kubectl-hepsre
	@echo "kubectl plugin installed to $(HOME)/.local/bin/kubectl-hepsre"
	@echo "Make sure $(HOME)/.local/bin is in your PATH"
	@echo "Usage: kubectl hepsre -namespace=<namespace> -pod=<pod>"

kubectl-plugin-uninstall: ## Uninstall kubectl plugin
	@echo "Uninstalling kubectl plugin..."
	@rm -f $(HOME)/.local/bin/kubectl-hepsre
	@echo "kubectl plugin uninstalled"

install-deps: ## Install Go dependencies
	go mod download
	go mod tidy

build: ## Build the server and CLI binaries
	go build -o bin/hep-sre-server cmd/server/main.go
	go build -o bin/hepsre cmd/cli/main.go

run: ## Run the server locally
	go run cmd/server/main.go

run-cli: ## Run the CLI (requires NAMESPACE and POD env vars)
	go run cmd/cli/main.go -namespace=$(NAMESPACE) -pod=$(POD) -lookback=$(LOOKBACK)

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/

docker-build: ## Build Docker image
	docker build -t micro-sre:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 \
		-v ~/.kube/config:/config \
		-e KUBECONFIG=/config \
		-e ANTHROPIC_API_KEY=$(ANTHROPIC_API_KEY) \
		micro-sre:latest

fmt: ## Format Go code
	go fmt ./...

lint: ## Run linter
	golangci-lint run

.DEFAULT_GOAL := help
