test: imports fix
	go test ./tests

test-v:
	go test ./tests -v

demo:
	go run example/main.go

bench:
	go test -bench="BenchmarkLimitOrders" -benchmem ./tests

imports:
	goimports -w .

fix:
	gofmt -s -w .
