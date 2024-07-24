VERSION ?= 0.0.4

GREEN='\033[0;32m'
NC='\033[0m'

GO_MOD_NAME = github.com/shoman4eg/go-moy-nalog

.PHONY: all init help test lint format version

all: format lint test

help: ## Show this help screen
	@printf 'Usage: make \033[36m<TARGETS>\033[0m ... \033[36m<OPTIONS>\033[0m\n\nAvailable targets are:'
	@awk 'BEGIN {FS = ":.*##"; printf "\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "    \033[36m%-17s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

init: ## Install required tools
	@echo -e ${GREEN}[Init]${NC}

	@cd tools && go mod tidy && go generate -x -tags=tools

test: ## Run unit TESTS
	@echo -e $(GREEN)[TEST]$(NC)
	@go test -v ./...

deps: ## Download required dependencies and remove unused
	@echo -e $(GREEN)[RESOLVE DEPENDENCIES]$(NC)
	go mod tidy

update: ## Update dependencies
	@echo -e $(GREEN)[UPDATE DEPENDENCIES]$(NC)
	go get -u

lint: ## Run required checkers and linters
	@echo -e ${GREEN}[Lint]${NC}

	@LOG_LEVEL=error bin/golangci-lint run
	@bin/go-consistent -pedantic ./...

FILES = $(shell find . -type f -name '*.go')

format: ## Format source code
	@echo -e ${GREEN}[Format]${NC}

	@bin/gofumpt -l -w $(FILES)
	@bin/goimports -local ${GO_MOD_NAME} -l -w $(FILES)
	@bin/gci write --section Standard --section Default --section "Prefix(${GO_MOD_NAME})" $(FILES)


version: ## Print Go version
	@echo -e $(GREEN)[VERSION]$(NC)
	@go version
