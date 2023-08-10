[![Go Report Card](https://goreportcard.com/badge/github.com/gonevo/matchingo)](https://goreportcard.com/report/github.com/gonevo/matchingo)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonevo/matchingo)
![GitHub](https://img.shields.io/github/license/gonevo/matchingo)

# Matchingo

Incredibly fast matching engine for HFT written in Golang

### Features

- supports **MARKET**, **LIMIT**, **STOP-LIMIT**, **OCO** order types
- supports _time-in-force_ (**GTK**, **FOK**, **IOC**) parameters for **LIMIT** orders
- does not use [shopspring/decimal](https://github.com/shopspring/decimal) for higher performance
- uses [lite decimal](https://github.com/nikolaydubina/fpdecimal) for price and quantity arguments
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

```shell
go get github.com/gonevo/matchingo
```

### Usage

### Decimal
Matchingo works with fpdecimal values, so feel free to use:

- `matchingo.FromInt(number int)`
- `matchingo.FromFloat(number float64)`

#### Order instance initialization parameters

- `matchingo.NewMarketOrder(orderID string, side Side, quantity fpdecimal.Decimal)`
- `matchingo.NewLimitOrder(orderID string, side Side, quantity, price fpdecimal.Decimal, tif TIF, oco string)`
- `matchingo.NewStopOrder(orderID string, side Side, quantity, price, stop fpdecimal.Decimal, oco string)`

> oco parameter is ID of another order from **OCO** orders set

#### Order processing

- `matchingo.Process(order *Order) (done *Done, err Error)`

#### Done instance

**Process()** returns **Done** instance which contains:

- **Order**: processed order
- **Trades**: slice of orders from the order book which are participants for this trade
    - **OrderID**: participant-order ID
    - **Price**: concrete trade price, string
    - **Quantity**: concrete trade quantity, string
    - **Role**: **TAKER** or **MAKER**, string
    - **IsQuote**: _true_ for **QUOTE quantity** orders
- **Canceled**: slice of order IDs which was cancelled for this processing (**IOC**, **OCO**), can be empty
- **Activated**: slice of order IDs which was activated for this processing (**STOP** orders), can be empty
- **Left**: _fpdecimal.Decimal_ value of left quantity for this processing, can be _fpdecimal.Zero_
- **Processed**: _fpdecimal.Decimal_ value of processed quantity for this processing, can be _fpdecimal.Zero_
- **Stored**: boolean, _true_ if order or its part was appended to **stop book** or **order book**

For example:

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

### Search
You can search your order at any time using ID

- `matchingo.GetOrder(id string) *Order`

> it returns an instance of Order or nil if Order not found

### Canceling
You can cancel your order at any time

- `matchingo.CancelOrder(id string) *Order`

> it returns an instance of Order if it was canceled or nil if Order not found

### Depth
You can see orderbook depth at any time

- `matchingo.Depth() map[string]string`
- `matchingo.DepthJSON() string`

For example:
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

> where key is a price, value is a volume 

### Example

```golang
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	orderBook := matchingo.NewOrderBook()
	matchingo.SetDecimalFraction(5) // default is 3

	done1, _ := orderBook.Process(matchingo.NewLimitOrder("order1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(10), "", ""))
	fmt.Println(done1)
	fmt.Println(orderBook)

	done2, _ := orderBook.Process(matchingo.NewLimitOrder("order2", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(10), "", ""))
	fmt.Println(done2)
	fmt.Println(orderBook)

	done3, _ := orderBook.Process(matchingo.NewMarketOrder("order3", matchingo.Buy, matchingo.FromFloat(5)))
	fmt.Println(done3)
	fmt.Println(orderBook)
}
```

### Benchmark

```shell
goos: linux
goarch: amd64
cpu: Intel(R) Core(TM) i7-8565U CPU @ 1.80GHz
BenchmarkLimitOrders-8 

Transactions per second (avg): 1139342.359570
 1000000              1755 ns/op            1011 B/op         14 allocs/op

```

### License

**matchingo** is open-sourced software licensed under the [MIT license](./LICENSE.md).

[Vano Devium](https://github.com/vanodevium/)

---

Made with ❤️ in Ukraine

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/vanodevium)
