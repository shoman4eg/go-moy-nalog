VERSION ?= 0.0.4

GREEN='\033[0;32m'
NC='\033[0m'

#
PKGS = $(shell go list ./...)

.PHONY: all init help clean test lint format version

all: format lint test

help: ## Show this help screen
	@printf 'Usage: make \033[36m<TARGETS>\033[0m ... \033[36m<OPTIONS>\033[0m\n\nAvailable targets are:'
	@awk 'BEGIN {FS = ":.*##"; printf "\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@printf "\nTargets run by default are: clean build format lint test.\n"

init: ## Install required tools
	@echo -e $(GREEN)[INIT]$(NC)
	@cd tools && go generate -x -tags=tools

test: ## Run unit TESTS
	@echo -e $(GREEN)[TEST]$(NC)
	@go test -v ./...

deps: ## Download required dependencies and remove unused
	@echo -e $(GREEN)[RESOLVE DEPENDENCIES]$(NC)
	go mod tidy

update: ## Update dependencies
	@echo -e $(GREEN)[UPDATE DEPENDENCIES]$(NC)
	go get -u

lint: ## Run linter on package sources
	@echo -e $(GREEN)[LINT]$(NC)
	@bin/golangci-lint run -v ./...

format: ## Format project sources
	@echo -e $(GREEN)[FORMAT]$(NC)
	@bin/gofumports -l -w .

version: ## Print Go version
	@echo -e $(GREEN)[VERSION]$(NC)
	@go version
