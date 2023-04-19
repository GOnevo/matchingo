[![Go Report Card](https://goreportcard.com/badge/github.com/gonevo/matchingo)](https://goreportcard.com/report/github.com/gonevo/matchingo)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonevo/matchingo)
![GitHub](https://img.shields.io/github/license/gonevo/matchingo)

# Matchingo

Incredibly fast matching engine for HFT written in Golang

### Features

- supports **MARKET**, **LIMIT**, **STOP-LIMIT**, **OCO** order types
- supports _time-in-force_ (**GTK**, **FOK**, **IOC**) parameters for **LIMIT** orders
- uses [shopspring/decimal](https://github.com/shopspring/decimal) for price and quantity arguments
- well tested code

##### Order quantity

> **It is very important!** Each SYMBOL is combination of BASE and QUOTE currencies.
> For example, **BTC/USD** where **BTC** is **BASE** currency, **USD** is **QUOTE** currency.

Each order **MUST HAVE** _quantity_ parameter but **pay attention**:
for **MARKET BUY** orders you have pass **QUOTE quantity**.

Examples for **BTC/USD** symbol:

- for **MARKET** **SELL** order 10 BTC, you have to pass quantity=10
- for **MARKET** **BUY** order 10 BTC, you have to pass maximum quantity of USD for attempt to buy 10 BTC

> This may seem inconvenient, but it is a precautionary measure for **double spending**: when a user places a
> **MARKET** **BUY** in asynchronous way, you will be able to freeze the correct quantity on his balance.

### Installation

```
go get github.com/gonevo/matchingo
```

### Usage

#### Order instance initialization parameters

- `NewMarketOrder(orderID string, side Side, quantity decimal.Decimal)`
- `NewLimitOrder(orderID string, side Side, quantity, price decimal.Decimal, tif TIF, oco string)`
- `NewStopOrder(orderID string, side Side, quantity, price, stop decimal.Decimal, oco string)`

> oco parameter is ID of another order from **OCO** orders set

#### Order processing

- `Process(order *Order) (done *Done, err Error)`

#### Done instance

**Process()** returns **Done** instance which contains:

- **Trade** instance
    - **Order**: exactly processed order
    - **Orders**: slice of orders from the order book which are participants for this trade
        - **OrderID**: participant-order ID
        - **Price**: concrete trade price, _decimal.Decimal_
        - **Quantity**: concrete trade quantity, _decimal.Decimal_
        - **ReferenceID**: reference to the exactly processed order
- **Canceled**: slice of order IDs which was cancelled for this processing (**IOC**, **OCO**), can be empty
- **Activated**: slice of order IDs which was activated for this processing (**STOP** orders), can be empty
- **Left**: _decimal.Decimal_ value of left quantity for this processing, can be _decimal.Zero_
- **Processed**: _decimal.Decimal_ value of processed quantity for this processing, can be _decimal.Zero_
- **Stored**: boolean, _true_ if order was appended to **stop book** or **order book**

### Example

```
package main

import (
	"fmt"
	"github.com/gonevo/matchingo"
	"github.com/shopspring/decimal"
)

func main() {
	orderBook := matchingo.NewOrderBook()
	fmt.Println(orderBook.Process(matchingo.NewLimitOrder("order-1", matchingo.Sell, decimal.New(10, 0), decimal.New(10, 0), "", "")))
	fmt.Println(orderBook.Process(matchingo.NewLimitOrder("order-2", matchingo.Buy, decimal.New(9, 0), decimal.New(10, 0), "", "")))
	fmt.Println(orderBook.Process(matchingo.NewMarketOrder("order-3", matchingo.Buy, decimal.New(10, 0))))
	fmt.Println(orderBook)
}
```

### Benchmark

```
goos: linux
goarch: amd64
cpu: Intel(R) Core(TM) i7-8565U CPU @ 1.80GHz
BenchmarkLimitOrders-8 

Elapsed: 1.176771449s
Transactions per second (avg): 5184983.035733

3050770               385.7 ns/op           544 B/op         10 allocs/op
```

### License

**matchingo** is open-sourced software licensed under the [MIT license](./LICENSE.md).

[Vano Devium](https://github.com/vanodevium/)

---

Made with ❤️ in Ukraine
