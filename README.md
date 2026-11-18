[![Go Report Card](https://goreportcard.com/badge/github.com/gonevo/matchingo)](https://goreportcard.com/report/github.com/gonevo/matchingo)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gonevo/matchingo)](https://github.com/gonevo/matchingo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# Matchingo

A fast, exact and recoverable order matching engine for exchanges, written in Go.

Matchingo is the heart of a trading venue: it takes buy and sell orders, pairs them by **price-time priority** (best price first, oldest order first), and tells you exactly what traded. It is built for low latency (about 75 ns and 2 allocations per order on an Apple M2) and uses exact fixed-point decimals, so there are no floating-point rounding surprises with money.

## Highlights

- **Order types**: limit, market, market-by-budget (quote), market-to-limit, stop-limit, stop-market, trailing stop, iceberg, post-only
- **Time-in-force**: GTC, IOC, FOK, and good-till-date (absolute expiry or TTL)
- **Protection**: OCO pairs, self-trade prevention, trading halts and a price circuit breaker
- **Amendments**: cancel, modify (keeps priority when only reducing size), atomic replace
- **Market rules**: tick size, lot size, quantity limits, price bands, decimal precision
- **Crash recovery**: snapshots plus a write-ahead log replay to the exact same state
- **Live feed**: trades, volume, best bid/offer, cancels, activations and halts as events
- **Concurrency**: single-threaded core with zero locking cost, plus an optional goroutine-safe wrapper
- **Deterministic**: same commands in, same trades and IDs out

## Installation

```shell
go get github.com/gonevo/matchingo
```

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Someone offers to sell 10 units at price 100.
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, matchingo.D(10), matchingo.D(100), "", ""))

	// Someone else wants to buy 4 units at 100: they trade immediately.
	done, _ := ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, matchingo.D(4), matchingo.D(100), "", ""))
	fmt.Println(done) // trades, what is left, whether the order rests on the book

	fmt.Println(ob.BestAsk()) // 6 units are still on offer at 100
}
```

Every `Process` call returns a `Done` report: which trades happened, what was canceled, which stop orders woke up, and what remains. Prices and quantities are `Decimal` values; build them with `matchingo.D(...)` (prefer strings like `D("99.5")` for fractions).

## Learn More

- **[PRODUCT.md](PRODUCT.md)**: how the engine works, in plain terms: every order type, matching rule, protection and recovery mechanism.
- **[examples/](examples/README.md)**: 23 small runnable programs, one concept each (limit orders, stops, iceberg, OCO, WAL recovery, concurrency, ...). Run one with:

```shell
go run ./examples/01_basic_limit_orders/
```

## Good to Know

- Use `NewConcurrentOrderBook` if several goroutines share one book. Scale by running one book per symbol.
- `order.SetCreatedAt(t)` supplies your own order timestamp, so the book skips its clock read for that order.
- Invalid input is rejected with a typed error (`ErrInvalidQuantity`, `ErrTradingHalted`, `ErrTickSize`, ...) and leaves the book untouched. Use `errors.Is` to check.
- Configuration (precision, instrument, circuit breaker) is set with options such as `WithPrecision`, `WithInstrument`, `WithCircuitBreaker`, `WithWAL`, `WithEmitter`, `WithClock`, and is not stored in snapshots: rebuild a recovered book with the same options.

## Running Tests

```shell
make test         # run all tests
make test-verbose # run tests with verbose output
make bench        # run benchmarks
make coverage     # show test coverage report
```

## Benchmark

```
goos: darwin
goarch: arm64
pkg: github.com/gonevo/matchingo/tests/orderbook
cpu: Apple M2
BenchmarkLimitOrders-8         3554872    318.4 ns/op    816 B/op    4 allocs/op
```

Each iteration rests one sell and fills it with one buy, with caller-supplied timestamps (`SetCreatedAt`).

## Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/my-improvement`)
3. **Commit** your changes (`git commit -am 'Add new feature'`)
4. **Push** to the branch (`git push origin feature/my-improvement`)
5. **Open** a Pull Request

Please ensure:

- All tests pass (`make test`)
- Code is formatted (`make fix`)
- New features include tests

## Security

Matchingo is intended for use in exchange backends where proper input validation and authorization should be handled by the surrounding application layer.

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email the maintainer at the repository's contact or open a private security advisory via GitHub
3. Allow reasonable time for a fix before public disclosure

This project follows standard responsible disclosure practices.

## License

**matchingo** is open-source software licensed under the [MIT License](./LICENSE.md).

---

Built by [Vano Devium](https://github.com/vanodevium/)

Made with ❤️ in Ukraine

[![Buy Me A Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/vanodevium)
