GO_MOD_NAME = github.com/shoman4eg/go-moy-nalog
GOLANGCI_LINT_VERSION ?= v2.10.1

GREEN='\033[0;32m'
NC='\033[0m'

.PHONY: all help init deps update format check-format lint test check cover version

all: format lint test

help: ## Show this help screen
	@printf 'Usage: make \033[36m<TARGETS>\033[0m ... \033[36m<OPTIONS>\033[0m\n\nAvailable targets are:'
	@awk 'BEGIN {FS = ":.*##"; printf "\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "    \033[36m%-17s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

init: ## Install required tools
	@echo -e ${GREEN}[Init]${NC}
	@rm -rf bin/*
	@curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b ./bin $(GOLANGCI_LINT_VERSION)

deps: ## Download required dependencies and remove unused
	@echo -e $(GREEN)[Resolve dependencies]$(NC)
	@go mod tidy

update: ## Update dependencies
	@echo -e $(GREEN)[Update dependencies]$(NC)
	@go get -u ./...
	@go mod tidy

FILES = $(shell find . -type f -name '*.go')

format: ## Format source code
	@echo -e ${GREEN}[Format]${NC}
	@go tool gofumpt -l -w $(FILES)
	@go tool goimports -local $(GO_MOD_NAME) -l -w $(FILES)
	@go tool gci write --section Standard --section Default --section "Prefix($(GO_MOD_NAME))" $(FILES)

lint: ## Run required checkers and linters
	@echo -e ${GREEN}[Lint]${NC}
	@LOG_LEVEL=error bin/golangci-lint run --config=./.golangci.yml ./...
	@go tool go-consistent -pedantic ./...

check-format: ## Report unformatted files without rewriting them
	@echo -e ${GREEN}[Check format]${NC}
	@out=$$(go tool gofumpt -l $(FILES)); \
		test -z "$$out" || { echo "gofumpt would rewrite:"; echo "$$out"; exit 1; }
	@out=$$(go tool goimports -local $(GO_MOD_NAME) -l $(FILES)); \
		test -z "$$out" || { echo "goimports would rewrite:"; echo "$$out"; exit 1; }
	@out=$$(go tool gci list --section Standard --section Default --section "Prefix($(GO_MOD_NAME))" $(FILES)); \
		test -z "$$out" || { echo "gci would rewrite:"; echo "$$out"; exit 1; }

test: ## Run tests
	@echo -e $(GREEN)[Test]$(NC)
	@go test -race -cover ./...

check: check-format lint test ## Verify everything without modifying any file
	@echo -e $(GREEN)[Check]$(NC)
	@go mod tidy -diff
	@echo -e $(GREEN)OK$(NC)

cover: ## Run tests and open the coverage report
	@echo -e $(GREEN)[Cover]$(NC)
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

version: ## Print Go version
	@echo -e $(GREEN)[Version]$(NC)
	@go version
