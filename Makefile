.DEFAULT_GOAL: help
.PHONY: help imports fix test test-v bench demo

help: ## Show this help message
	@sed -rn 's/^([a-zA-Z_-]+):.*## (.*)$$/"\1" "\2"/p' < $(MAKEFILE_LIST) | xargs printf "make %-20s# %s\n"

imports: ## Run goimports to tidy imports
	goimports -w .
fix: ## Run gofmt to fix formatting
	gofmt -s -w .

test: imports fix ## Run tests
	go test ./tests/...
test-v: imports fix ## Run tests with verbose output
	go test ./tests/... -v
bench: ## Run benchmarks
	go test -bench="BenchmarkLimitOrders" -benchmem ./tests/...

demo: ## Run the example
	go run example/main.go
