# Colors
GREEN_COLOR   = "\033[0;32m"
PURPLE_COLOR  = "\033[0;35m"
DEFAULT_COLOR = "\033[m"

.PHONY: help deps

help: ## This help message
	@printf 'Usage: make \033[36m<TARGETS>\033[0m ... \033[36m<OPTIONS>\033[0m\n\nAvailable targets are:'
	@awk 'BEGIN {FS = ":.*##"; printf "\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "    \033[36m%-17s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@printf "\nTargets run by default are: clean format lint vet coverage test.\n"

deps: ## Download reqired dependencies and remove unused.
	@echo -e $(GREEN_COLOR)[RESOLVE DEPENDENCIES]$(DEFAULT_COLOR)
	go mod tidy

update: ## Update dependencies.
	@echo -e $(GREEN_COLOR)[UPDATE DEPENDENCIES]$(DEFAULT_COLOR)
	go get -u ./v1
