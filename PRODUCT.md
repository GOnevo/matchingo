# matchingo — Order Book Matching Engine

A Go package implementing a high-performance, price-time priority order book matching engine for financial exchanges and trading systems.

---

## Overview

`matchingo` provides a complete limit order book with market, limit, and stop-limit order types, multiple time-in-force policies, iceberg orders, post-only orders, OCO (one-cancels-the-other) pairs, snapshot/restore, and an event emitter interface for real-time feed consumers.

The engine uses a sorted slice per side for price-level management, giving O(1) best-price lookup and cache-friendly traversal, while FIFO order queues at each price level enforce time priority.

---

## Core Types

### `OrderBook`

The root object. Holds the bid side, ask side, stop book, OCO registry, and an optional event emitter.

```go
ob := matchingo.NewOrderBook()
ob.SetQuantityPrecision(2) // optional: reject quantities with more than 2 decimal places
ob.SetPricePrecision(4)    // optional: reject prices with more than 4 decimal places
```

### `Order`

An immutable-ish value that carries all order attributes. Created through dedicated constructors; never mutated directly by callers.

### `Done`

The result of a `Process` call. Contains:
- `Order` — the submitted order (as `TradeOrder`)
- `Trades` — every fill that occurred, with maker/taker roles, price, and amount
- `Canceled` — IDs of orders canceled as a side effect (OCO partner, IOC remainder, etc.)
- `Activated` — IDs of stop-limit orders that were triggered
- `Activations` — `Done` results from each activated stop order's execution
- `Left` — unfilled quantity remaining
- `Processed` — quantity that traded
- `Stored` — whether the order was placed on the book as a resting order

---

## Order Types

### Market Order

Executes immediately against the best available prices. Unfilled quantity is canceled automatically.

```go
order := matchingo.NewMarketOrder("id1", matchingo.Buy, qty)
```

### Market Quote Order

Same as a market order, but `quantity` is denominated in the **quote currency** (a spend budget). The engine converts the budget to base units at each price level, so the caller spends at most `quantity` quote.

```go
order := matchingo.NewMarketQuoteOrder("id2", matchingo.Buy, budget)
```

### Limit Order

Rests on the book at the specified price if it cannot be fully filled immediately. Accepts a TIF policy and an optional OCO reference.

```go
order := matchingo.NewLimitOrder("id3", matchingo.Sell, qty, price, matchingo.GTC, "")
```

### Stop-Limit Order

Sits in the stop book until a trade at or through the stop price triggers it, at which point it becomes a limit order and is processed normally.

```go
order := matchingo.NewStopLimitOrder("id4", matchingo.Buy, qty, limitPrice, stopPrice, "")
```

Stop activation rules:
- **Buy** stops fire when `trade price ≥ stop price` (lowest stop prices first)
- **Sell** stops fire when `trade price ≤ stop price` (highest stop prices first)

### Post-Only Order

A limit order that is rejected (canceled) if it would match immediately. Guarantees the order will only ever rest as a maker.

```go
order := matchingo.NewPostOnlyOrder("id5", matchingo.Buy, qty, price, matchingo.GTC, "")
```

### Iceberg Order

A large limit order that shows only a `peakQty` slice on the book. When the visible slice is filled, the next slice is automatically re-queued at the **back** of the same price level, hiding the total size from the market.

```go
order := matchingo.NewIcebergOrder("id6", matchingo.Sell, totalQty, peakQty, price, matchingo.GTC, "")
```

---

## Time-in-Force (TIF)

| Value | Behavior |
|-------|----------|
| `GTC` | Good Till Canceled — rests until filled or explicitly canceled (default) |
| `IOC` | Immediate Or Cancel — fills what it can; cancels the remainder immediately |
| `FOK` | Fill Or Kill — fills entirely in one shot or is canceled entirely; never rests |

---

## OCO (One-Cancels-the-Other)

Two orders can be linked by setting each other's OCO field to the partner's ID. When one of the pair is filled (or partially filled and removed), the other is automatically canceled.

Supported combinations:
- Two limit orders
- A limit order paired with a stop-limit order

---

## Operations

### Submit an Order

```go
done, err := ob.Process(order)
```

Returns a `*Done` with full trade and state details, or an error for invalid input.

### Cancel an Order

```go
ob.Cancel(orderID)          // returns bool
ob.CancelOrder(orderID)     // returns *Order
```

Removes the order from the limit book or the stop book. Emits `EventCanceled`.

### Modify a Resting Limit Order

```go
done, ok := ob.Modify(orderID, newPrice, newQty)
```

Two paths:
- **In-place decrease** — same price and `newQty ≤ current open` → quantity reduced without losing time priority
- **Re-queue** — price change or quantity increase → order is removed and re-inserted at the back of the new price level, then matched as a fresh limit order

Stop-limit orders cannot be modified.

### Look Up an Order

```go
order := ob.GetOrder(orderID)
```

### Best Bid / Ask

```go
level := ob.BestBid()  // *Level{Price, Volume}
level := ob.BestAsk()
```

### Volume at a Price Level

```go
vol := ob.Level(matchingo.Buy, price)   // *Decimal, nil if no orders there
```

### Market Price Estimate

```go
price, err := ob.CalculateMarketPrice(matchingo.Buy, quantity)
```

Walks the opposite side's price levels and returns the total cost to fill `quantity` base units at current resting prices. Returns `ErrInsufficientQuantity` if the book lacks enough volume.

### Depth Snapshot

```go
depth := ob.Depth()       // *Depth{Ask map[string]string, Bid map[string]string}
json  := ob.DepthJSON()   // JSON string
```

---

## Event System

Attach any implementation of `EventEmitter` to receive a real-time feed of engine activity:

```go
type EventEmitter interface {
    Emit(eventType EventType, data any)
}

ob.SetEmitter(myEmitter)
```

**Channel emitter (async)** — a built-in implementation sends events to a `chan<- Event` you own and read in a separate goroutine:

```go
ch := make(chan matchingo.Event, 1024)
ob.SetEventChannel(ch, true) // true = drop on full (non-blocking); false = backpressure

go func() {
    for e := range ch {
        switch e.Type {
        case matchingo.EventTrade:
            d := e.Data.(matchingo.TradeEventData)
            // ...
        }
    }
}()
```

| Event | `data` type | Fired when |
|-------|-------------|------------|
| `EventTrade` | `TradeEventData{OrderID, Price, Amount, Role}` | A fill occurs (once per maker, once per taker) |
| `EventVolume` | `VolumeEventData{Side, Price, Volume}` | Total volume at a price level changes |
| `EventDepth` | `*Depth` | `ob.Depth()` is called |
| `EventCanceled` | `string` (order ID) | Any order is canceled for any reason |
| `EventActivated` | `string` (order ID) | A stop-limit order is triggered and executed |

---

## Snapshot / Restore

The full engine state can be serialized to a plain-data struct and restored later, enabling persistence, replication, or crash recovery.

```go
snap := ob.Dump()           // Snapshot{Version, Bids, Asks, Stops, OCO}
err  := ob.Restore(snap)    // clears state and reloads from snap
```

`Snapshot` is version-stamped; `Restore` returns an error if the version does not match the current engine format. All order attributes — including iceberg hidden quantities, OCO links, and stop prices — are preserved.

---

## Errors

| Error | Meaning |
|-------|---------|
| `ErrInvalidOrder` | Nil or unrecognized order type |
| `ErrInvalidSide` | Side is neither Buy nor Sell |
| `ErrCanceledOrder` | Order is already canceled |
| `ErrInvalidQuantity` | Quantity ≤ 0 |
| `ErrInvalidPrice` | Price ≤ 0 |
| `ErrInvalidTif` | Unrecognized TIF value |
| `ErrOrderExists` | Duplicate order ID |
| `ErrInsufficientQuantity` | Not enough volume for `CalculateMarketPrice` |
| `ErrQuantityPrecision` | Quantity has more decimal places than the book's `SetQuantityPrecision` limit |
| `ErrPricePrecision` | Price has more decimal places than the book's `SetPricePrecision` limit |

---

## Internal Architecture

| Component | Responsibility |
|-----------|---------------|
| `OrderSide` | One side of the book (bids or asks). Maintains a sorted `[]Decimal` of price levels for O(1) best-price lookup and cache-friendly traversal. |
| `OrderQueue` | FIFO queue of orders at a single price level, plus the total volume at that level. |
| `StopBook` | Hash map of stop prices → queues. On each trade, scans for triggered stops and returns activated orders sorted by proximity to the trade price. |
| `Decimal` | Arbitrary-precision decimal arithmetic wrapping `shopspring/decimal`, keeping all price/quantity math exact. |

---

## Package

```
module: github.com/i25959341/orderbook
import: matchingo
```
