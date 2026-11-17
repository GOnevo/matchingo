[![Go Report Card](https://goreportcard.com/badge/github.com/gonevo/matchingo)](https://goreportcard.com/report/github.com/gonevo/matchingo)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gonevo/matchingo)](https://github.com/gonevo/matchingo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Test Coverage](https://gocov.io/?package=github.com/gonevo/matchingo)](https://gocov.io/?package=github.com/gonevo/matchingo)

# Matchingo

High-performance order matching engine for cryptocurrency exchanges, written in Go.

Matchingo implements a standard price-time priority matching algorithm with support for market, limit, and stop-limit orders, as well as OCO (One-Cancels-Other) order pairs. It is designed for high-frequency trading (HFT) scenarios where low latency and minimal allocations are critical.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Examples](#examples)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
  - [Decimal](#decimal)
  - [Creating Orders](#creating-orders)
  - [Processing Orders](#processing-orders)
  - [Done Result](#done-result)
  - [Order Lookup](#order-lookup)
  - [Canceling Orders](#canceling-orders)
  - [Order Book Depth](#order-book-depth)
  - [Market Price Calculation](#market-price-calculation)
  - [Precision Limits](#precision-limits)
- [Order Quantity & Quote Mode](#order-quantity--quote-mode)
- [Time in Force](#time-in-force)
- [OCO Orders](#oco-orders)
- [Stop-Limit Orders](#stop-limit-orders)
- [Events](#events)
- [Error Handling](#error-handling)
- [Running Tests](#running-tests)
- [Benchmark](#benchmark)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

## Features

- **Order types**: `MARKET`, `LIMIT`, `STOP-LIMIT`
- **Time-in-force**: `GTC` (Good Till Cancel), `FOK` (Fill or Kill), `IOC` (Immediate or Cancel)
- **OCO (One-Cancels-Other)**: pair orders where filling one cancels the other
- **Quote-mode market orders**: specify market buy quantity in quote currency for accurate balance freezing
- **High performance**: uses [fpdecimal](https://github.com/nikolaydubina/fpdecimal) instead of [shopspring/decimal](https://github.com/shopspring/decimal) for lower allocation overhead
- **Well tested**: comprehensive test suite covering order book, stop book, order queue, and decimal operations

## Installation

```shell
go get github.com/gonevo/matchingo
```

## Examples

The [`examples/`](examples/) directory contains runnable examples, one per
concept, each with inline explanations:

| # | Topic | File |
|---|-------|------|
| 01 | Basic limit orders | [examples/01_basic_limit_orders/](examples/01_basic_limit_orders/main.go) |
| 02 | Market & quote-market orders | [examples/02_market_orders/](examples/02_market_orders/main.go) |
| 03 | Time-in-force (GTC / IOC / FOK) | [examples/03_time_in_force/](examples/03_time_in_force/main.go) |
| 04 | Cancel and modify | [examples/04_cancel_and_modify/](examples/04_cancel_and_modify/main.go) |
| 05 | Stop-limit orders | [examples/05_stop_limit_orders/](examples/05_stop_limit_orders/main.go) |
| 06 | OCO (One-Cancels-Other) | [examples/06_oco_orders/](examples/06_oco_orders/main.go) |
| 07 | Iceberg orders | [examples/07_iceberg_orders/](examples/07_iceberg_orders/main.go) |
| 08 | Post-only orders | [examples/08_post_only_orders/](examples/08_post_only_orders/main.go) |
| 09 | Snapshot (dump & restore) | [examples/09_snapshot/](examples/09_snapshot/main.go) |
| 10 | Event emitter | [examples/10_event_emitter/](examples/10_event_emitter/main.go) |
| 11 | Precision limits | [examples/11_precision_limits/](examples/11_precision_limits/main.go) |
| 12 | Channel emitter (async) | [examples/12_channel_emitter/](examples/12_channel_emitter/main.go) |
| 13 | Sequence numbers | [examples/13_sequence/](examples/13_sequence/main.go) |

Run any example with:

```shell
go run ./examples/01_basic_limit_orders/
```

See [examples/README.md](examples/README.md) for a full description of each.

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Place a limit sell order: 10 units at price 10
	done1, _ := ob.Process(matchingo.NewLimitOrder(
		"order1", matchingo.Sell,
		matchingo.FromInt(10), matchingo.FromInt(10),
		"", "",
	))
	fmt.Println(done1)

	// Place a limit buy order: 5 units at price 10 (will match with order1)
	done2, _ := ob.Process(matchingo.NewLimitOrder(
		"order2", matchingo.Buy,
		matchingo.FromInt(5), matchingo.FromInt(10),
		"", "",
	))
	fmt.Println(done2)

	// Place a market buy order
	done3, _ := ob.Process(matchingo.NewMarketOrder(
		"order3", matchingo.Buy,
		matchingo.FromFloat(5),
	))
	fmt.Println(done3)
}
```

See [`example/main.go`](example/main.go) for a runnable example, or run:

```shell
make example
```

## API Reference

### Decimal

Matchingo uses a fixed-point `Decimal` type built on [fpdecimal](https://github.com/nikolaydubina/fpdecimal) for precise price and quantity arithmetic. **Never use `float64` literals directly for prices or quantities** — binary floating-point cannot represent values like `0.1` exactly, which causes silent rounding errors in financial arithmetic. Use integer or string constructors instead.

The quickest way to create a Decimal from any Go numeric type or string is the generic helper `D`:

```go
matchingo.D(100)      // integer — exact
matchingo.D("99.5")   // string  — preferred for fractional values
matchingo.D(99.5)     // float64 — only use when the value is exactly representable
```

| Function | Description |
|---|---|
| `matchingo.D[T](v T)` | Generic helper: accepts int, float, or string; panics on bad string input |
| `matchingo.FromInt(n int)` | Create a Decimal from an integer |
| `matchingo.FromFloat(f float64)` | Create a Decimal from a float64 |
| `matchingo.FromString(s string)` | Parse a Decimal from a string (returns error) |
| `matchingo.FromIntScaled(n int64)` | Create a Decimal from a scaled integer |
| `matchingo.Zero` | Zero value constant |

Decimal supports arithmetic operations: `Add`, `Sub`, `Mul`, `Div`, and comparisons: `Equal`, `LessThan`, `LessThanOrEqual`, `GreaterThan`, `GreaterThanOrEqual`, `Compare`.

### Creating Orders

| Constructor | Description |
|---|---|
| `NewMarketOrder(id string, side Side, quantity Decimal)` | Market order (base quantity) |
| `NewMarketQuoteOrder(id string, side Side, quantity Decimal)` | Market order with quantity in quote currency |
| `NewLimitOrder(id string, side Side, quantity, price Decimal, tif TIF, oco string)` | Limit order with optional TIF and OCO |
| `NewStopLimitOrder(id string, side Side, quantity, price, stop Decimal, oco string)` | Stop-limit order with optional OCO |

**Parameters:**

- `id` — unique order identifier
- `side` — `matchingo.Buy` or `matchingo.Sell`
- `quantity` — order quantity (base currency, unless quote mode)
- `price` — limit price (for limit and stop-limit orders)
- `stop` — stop price (for stop-limit orders)
- `tif` — time-in-force: `""` (GTC), `matchingo.GTC`, `matchingo.FOK`, or `matchingo.IOC`
- `oco` — ID of the paired OCO order (empty string if not paired)

### Processing Orders

Every `OrderBook` instance processes orders independently:

```go
ob := matchingo.NewOrderBook()
done, err := ob.Process(order)
```

`Process` accepts `MARKET`, `LIMIT`, and `STOP-LIMIT` orders. It returns a `*Done` result and an error if the order is invalid.

### Done Result

`Process` returns a `*Done` struct containing the full trade outcome:

| Field | Type | Description |
|---|---|---|
| `Order` | `*Order` | The processed order |
| `Trades` | `[]TradeOrder` | Trade details for each matched pair |
| `Canceled` | `[]string` | Order IDs canceled during processing (IOC, OCO) |
| `Activated` | `[]string` | Stop order IDs activated during processing |
| `Stored` | `bool` | `true` if the order (or remainder) was stored in the book |
| `Quantity` | `Decimal` | Original order quantity |
| `Left` | `Decimal` | Remaining unfilled quantity |
| `Processed` | `Decimal` | Total filled quantity |

Each `TradeOrder` in `Trades` contains:

| Field | Type | Description |
|---|---|---|
| `OrderID` | `string` | Participant order ID |
| `Role` | `Role` | `MAKER` or `TAKER` |
| `Price` | `Decimal` | Execution price |
| `Quantity` | `Decimal` | Trade quantity |
| `IsQuote` | `bool` | `true` if quantity is in quote currency |

`Done` implements `MarshalJSON` and `String` for easy serialization:

```json
{
  "order": {
    "orderID": "order2",
    "role": "TAKER",
    "price": "10.00000",
    "isQuote": false,
    "quantity": "9.00000"
  },
  "trades": [
    {
      "orderID": "order2",
      "role": "TAKER",
      "price": "10.00000",
      "isQuote": false,
      "quantity": "9.00000"
    },
    {
      "orderID": "order1",
      "role": "MAKER",
      "price": "10.00000",
      "isQuote": false,
      "quantity": "9.00000"
    }
  ],
  "canceled": [],
  "activated": [],
  "left": "0",
  "processed": "9.00000",
  "stored": false
}
```

### Order Lookup

Retrieve an order by its ID:

```go
order := ob.GetOrder("order1") // returns *Order or nil
```

### Canceling Orders

Cancel an order by its ID:

```go
order := ob.CancelOrder("order1") // returns *Order if canceled, nil if not found
```

This works for orders in both the order book and the stop book.

### Order Book Depth

Get the current bid/ask depth:

```go
depth := ob.Depth()      // *Depth struct with Ask and Bid maps
jsonStr := ob.DepthJSON() // JSON string representation
```

Example output:

```json
{
  "ask": {
    "21.00000": "20.00000",
    "22.00000": "10.00000",
    "23.00000": "20.00000"
  },
  "bid": {
    "10.00000": "20.00000",
    "11.00000": "10.00000",
    "12.00000": "20.00000"
  }
}
```

Where each key is a price level and each value is the total volume at that level.

### Market Price Calculation

Estimate the total cost to fill a market order of a given size:

```go
price, err := ob.CalculateMarketPrice(matchingo.Buy, matchingo.FromInt(100))
```

Returns `ErrInsufficientQuantity` if the order book cannot fully fill the requested quantity.

## Precision Limits

An `OrderBook` instance can enforce maximum decimal-place limits on quantities and prices. Orders that exceed the limit are rejected by `Process` with a typed error — no panics, no silent truncation.

Set precision at construction with options (preferred):

```go
ob := matchingo.NewOrderBook(matchingo.WithPrecision(2, 4))
// or individually:
ob := matchingo.NewOrderBook(
    matchingo.WithQuantityPrecision(2),
    matchingo.WithPricePrecision(4),
)
```

Or on an existing book with setter methods:

```go
ob := matchingo.NewOrderBook()
ob.SetQuantityPrecision(2) // reject quantities with more than 2 decimal places
ob.SetPricePrecision(4)    // reject prices with more than 4 decimal places

// accepted: quantity 1.25, price 10.1234
done, err := ob.Process(matchingo.NewLimitOrder(
    "order1", matchingo.Buy,
    matchingo.FromFloat(1.25), matchingo.FromFloat(10.1234),
    "", "",
))

// rejected: quantity 1.251 has 3 decimal places → ErrQuantityPrecision
_, err = ob.Process(matchingo.NewLimitOrder(
    "order2", matchingo.Buy,
    matchingo.FromFloat(1.251), matchingo.FromFloat(10.1234),
    "", "",
))
// err == matchingo.ErrQuantityPrecision
```

A precision of `0` (the default) means no limit. The check applies to the quantity and limit/stop price of every order type.

## Order Quantity & Quote Mode

Each trading symbol combines a **BASE** and a **QUOTE** currency (e.g., `BTC/USD` where `BTC` is "base" and `USD` is "quote").

For **MARKET SELL** orders, the quantity is in the base currency:

```go
// Sell 10 BTC
ob.Process(matchingo.NewMarketOrder("sell1", matchingo.Sell, matchingo.FromInt(10)))
```

For **MARKET BUY** orders, you can use quote-mode to specify the maximum quote currency to spend:

```go
// Buy BTC with up to 1000 USD
ob.Process(matchingo.NewMarketQuoteOrder("buy1", matchingo.Buy, matchingo.FromInt(1000)))
```

This is a safety measure against double spending: when placing a market buy asynchronously, you can freeze the exact quote amount on the user's balance before execution.

## Time in Force

Limit orders support three time-in-force modes:

| TIF | Behavior |
|---|---|
| `GTC` (Good Till Cancel) | Order remains in the book until fully filled or manually canceled (default) |
| `FOK` (Fill or Kill) | Order is canceled immediately if it cannot be filled in its entirety |
| `IOC` (Immediate or Cancel) | Order fills as much as possible immediately; the remainder is canceled |

```go
// Fill-or-kill limit order
ob.Process(matchingo.NewLimitOrder(
	"order1", matchingo.Buy,
	matchingo.FromInt(10), matchingo.FromInt(100),
	matchingo.FOK, "",
))
```

## OCO Orders

OCO (One-Cancels-Other) pairs two orders so that filling one automatically cancels the other. Pass the paired order's ID as the `oco` parameter:

```go
orderA := matchingo.NewLimitOrder("a", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), "", "b")
orderB := matchingo.NewLimitOrder("b", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(90), "", "a")

ob.Process(orderA)
ob.Process(orderB)
// If orderA fills, orderB is automatically canceled (and vice versa)
```

## Stop-Limit Orders

A stop-limit order sits in the stop book until the market price reaches the stop price, at which point it becomes a limit order:

```go
stopOrder := matchingo.NewStopLimitOrder(
	"stop1", matchingo.Sell,
	matchingo.FromInt(5),   // quantity
	matchingo.FromInt(95),  // limit price
	matchingo.FromInt(90),  // stop price
	"",                     // oco
)
ob.Process(stopOrder)
```

When the stop price is touched during matching, the order activates and enters the order book as a limit order. Activated order IDs appear in the `Done.Activated` slice.

## Events

The order book can push real-time events to any `EventEmitter`. Attach one at construction or via `SetEmitter`:

```go
// at construction (preferred)
ob := matchingo.NewOrderBook(matchingo.WithEmitter(&myLogger{}))

// or after construction
ob.SetEmitter(&myLogger{})

// built-in async channel emitter
ch := make(chan matchingo.Event, 64)
ob := matchingo.NewOrderBook(
    matchingo.WithEmitter(matchingo.NewChannelEmitter(ch, true /* drop on full */)),
)
```

Implement `EventEmitter` to receive events:

```go
type EventEmitter interface {
    Emit(eventType EventType, data any)
}
```

| Event type | Data type | Fired when |
|---|---|---|
| `EventTrade` | `TradeEventData` | A fill occurs |
| `EventVolume` | `VolumeEventData` | Volume at a price level changes |
| `EventCanceled` | `string` (order ID) | An order is canceled |
| `EventActivated` | `string` (order ID) | A stop-limit order activates |
| `EventBBO` | `BBOEventData` | Best bid or ask changes |

`ChannelEmitter` also stamps each `Event.Sequence` with the book's monotonic counter. See [example 10](examples/10_event_emitter/main.go) and [example 12](examples/12_channel_emitter/main.go).

## Error Handling

`Process` returns these errors for invalid input:

| Error | Condition |
|---|---|
| `ErrInvalidOrder` | Order is nil or not MARKET/LIMIT/STOP-LIMIT |
| `ErrInvalidSide` | Side is neither Buy nor Sell |
| `ErrCanceledOrder` | Order has already been canceled |
| `ErrInvalidQuantity` | Quantity is zero or negative |
| `ErrInvalidPrice` | Price is zero or negative |
| `ErrInvalidTif` | TIF value is not GTC, FOK, or IOC |
| `ErrOrderExists` | An order with the same ID already exists |
| `ErrInsufficientQuantity` | Not enough volume to calculate market price |
| `ErrQuantityPrecision` | Quantity has more decimal places than allowed by `SetQuantityPrecision` |
| `ErrPricePrecision` | Price has more decimal places than allowed by `SetPricePrecision` |

Order constructors (`NewMarketOrder`, `NewLimitOrder`, `NewStopLimitOrder`) panic on invalid quantity or price.

## Running Tests

```shell
make test         # run all tests
make test-verbose # run tests with verbose output
make bench        # run benchmarks
make coverage     # show test coverage report
make example      # run the example
```

## Benchmark

```
goos: darwin
goarch: arm64
pkg: github.com/gonevo/matchingo/tests/orderbook
cpu: Apple M2
BenchmarkLimitOrders-8    15819102    73.58 ns/op    288 B/op    2 allocs/op
```

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
