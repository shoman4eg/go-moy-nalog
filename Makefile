SHELL := /bin/bash
MAKEFILE_PATH := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
PATH := $(MAKEFILE_PATH):$(PATH)

#
PKGS = $(shell go list ./...)

# Colors
GREEN_COLOR   = "\033[0;32m"
PURPLE_COLOR  = "\033[0;35m"
DEFAULT_COLOR = "\033[m"

.PHONY: all init help clean test lint format build run install version

all: clean build format lint test

help: ## Show this help screen
	@printf 'Usage: make \033[36m<TARGETS>\033[0m ... \033[36m<OPTIONS>\033[0m\n\nAvailable targets are:'
	@awk 'BEGIN {FS = ":.*##"; printf "\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "    \033[36m%-17s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@printf "\nTargets run by default are: clean build format lint test.\n"

init: ## Install required tools
	@echo -e $(GREEN_COLOR)[INIT]$(DEFAULT_COLOR)
	@cd tools && go generate -x -tags=tools

clean: ## Remove binary
	@echo -e $(GREEN_COLOR)[CLEAN]$(DEFAULT_COLOR)
	@$(GOCLEAN)
	@if [ -f $(BINARY) ] ; then rm $(BINARY) ; fi

test: ## Run unit TESTS
	@echo -e $(GREEN_COLOR)[TEST]$(DEFAULT_COLOR)
	@go test -race $(PKGS)

deps: ## Download required dependencies and remove unused
	@echo -e $(GREEN_COLOR)[RESOLVE DEPENDENCIES]$(DEFAULT_COLOR)
	go mod tidy

update: ## Update dependencies
	@echo -e $(GREEN_COLOR)[UPDATE DEPENDENCIES]$(DEFAULT_COLOR)
	go get -u

lint: ## Run linter on package sources
	@echo -e $(GREEN_COLOR)[LINT]$(DEFAULT_COLOR)
	@bin/golangci-lint run --config=$(MAKEFILE_PATH)/.golangci.yml

format: ## Format project sources
	@echo -e $(GREEN_COLOR)[FORMAT]$(DEFAULT_COLOR)
	@bin/gofumports -l -w .

run: ## Compile and run Go program
	@echo -e $(GREEN_COLOR)[RUN]$(DEFAULT_COLOR)
	@go run -race main.go

install: ## Compile and install packages and dependencies
	@echo -e $(GREEN_COLOR)[INSTALL]$(DEFAULT_COLOR)
	@go install install $(LDFLAGS) ./...

version: ## Print Go version
	@echo -e $(GREEN_COLOR)[VERSION]$(DEFAULT_COLOR)
	@go version
