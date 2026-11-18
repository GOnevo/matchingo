.DEFAULT_GOAL: help
.PHONY: help imports fix test test-verbose coverage bench

help: ## Show this help message
	@sed -rn 's/^([a-zA-Z_-]+):.*## (.*)$$/"\1" "\2"/p' < $(MAKEFILE_LIST) | xargs printf "make %-20s# %s\n"

imports: ## Run goimports to tidy imports
	goimports -w .
fix: ## Run gofmt to fix formatting
	gofmt -s -w .

test: imports fix ## Run tests
	go test ./tests/...
test-verbose: imports fix ## Run tests with verbose output
	go test ./tests/... -v
coverage: ## Show test coverage table
	go test -coverpkg=github.com/gonevo/matchingo -coverprofile=/tmp/cover.out ./tests/... >/dev/null 2>&1 && go tool cover -func=/tmp/cover.out
bench: ## Run benchmarks (1 second each)
	go test -run=^$$ -bench="BenchmarkLimitOrders" -benchtime=1s -benchmem ./tests/orderbook/
