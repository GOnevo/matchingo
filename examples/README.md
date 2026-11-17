# Matchingo Examples

Runnable examples that show every major feature of the engine, from the
simplest limit order to stop-limit activations and event streaming.

Each example lives in its own directory and can be run with `go run`:

```shell
go run ./examples/01_basic_limit_orders/
```

---

## 01 · Basic Limit Orders

**File:** [01_basic_limit_orders/main.go](01_basic_limit_orders/main.go)

The entry point.  Shows how to create an order book, place GTC limit orders on
both sides, and read the `Done` result to see what was matched.

Key ideas:
- `NewLimitOrder` with `GTC` time-in-force
- `done.Stored` → order rests in the book
- `done.Trades` → slice of fills; `[0]` is taker, `[1+]` are makers
- Partial fills: `done.Processed` + `done.Left`

---

## 02 · Market Orders

**File:** [02_market_orders/main.go](02_market_orders/main.go)

Market orders execute at the best available price without a price limit.  When
the opposite side is exhausted `done.Left` shows the unfilled remainder.

Also covers **quote-mode** market orders (`NewMarketQuoteOrder`) where you
specify how much of the *quote* currency to spend rather than a base quantity.

Key ideas:
- `NewMarketOrder` — quantity in base currency
- `NewMarketQuoteOrder` — quantity in quote currency (e.g. "spend $400")
- `done.Left > 0` when liquidity runs out

---

## 03 · Time-in-Force (GTC / IOC / FOK)

**File:** [03_time_in_force/main.go](03_time_in_force/main.go)

Controls what happens to the unfilled part of an order.

| TIF | Behaviour |
|-----|-----------|
| `GTC` | Rests in the book until canceled (default) |
| `IOC` | Fills what it can; cancels the remainder immediately |
| `FOK` | Must fill completely or the whole order is rejected |

Key ideas:
- `matchingo.GTC`, `matchingo.IOC`, `matchingo.FOK` constants
- IOC: `done.Stored=false`, `done.Left > 0` on partial fill
- FOK: zero trades and `done.Stored=false` on rejection

---

## 04 · Cancel and Modify

**File:** [04_cancel_and_modify/main.go](04_cancel_and_modify/main.go)

Orders resting in the book can be removed entirely or adjusted.

**Modify rules:**
- *Quantity decrease, same price* → reduced in-place, keeps time priority
- *Price change or quantity increase* → re-queued at the back of the new
  price level, then matched (loses time priority)

Key ideas:
- `ob.Cancel(orderID) bool`
- `ob.Modify(orderID, newPrice, newQty) (*Done, bool)`
- Re-queue path may immediately trigger a match

---

## 05 · Stop-Limit Orders

**File:** [05_stop_limit_orders/main.go](05_stop_limit_orders/main.go)

Stop-limit orders sleep in the **stop book** until a live trade crosses their
stop price, then activate as regular limit orders.

**Direction rule:**
- Sell stop → activates when market trades at or *below* the stop price
- Buy  stop → activates when market trades at or *above* the stop price

Key ideas:
- `NewStopLimitOrder(id, side, qty, limitPrice, stopPrice, oco)`
- Stop orders are invisible in the live book depth
- `done.Activated` — IDs of orders that became live
- `done.Activations` — `*Done` result of each activated order's match attempt

---

## 06 · OCO (One-Cancels-Other) Orders

**File:** [06_oco_orders/main.go](06_oco_orders/main.go)

Links two orders so that when one executes, the engine automatically cancels
the other.  Classic use-case: take-profit + stop-loss on a resting position.

**Linkage is bidirectional by order ID:**
```
orderA  oco = "orderB"
orderB  oco = "orderA"
```

Key ideas:
- The `oco` parameter in `NewLimitOrder` / `NewStopLimitOrder` is the sibling's **order ID**
- `done.Canceled` — IDs auto-canceled by OCO logic
- Works when sibling is a resting limit *or* a pending/activated stop

---

## 07 · Iceberg Orders

**File:** [07_iceberg_orders/main.go](07_iceberg_orders/main.go)

An iceberg order hides most of its volume.  Only the "peak" slice is visible
at any time; each time the peak is consumed the next slice is revealed
automatically until the total quantity is exhausted.

Useful for large institutional orders that must not signal their size.

Key ideas:
- `NewIcebergOrder(id, side, totalQty, peakQty, price, tif, oco)`
- Book depth shows only the peak, not the total hidden volume
- Peak replenishment is fully automatic — no caller action needed

---

## 08 · Post-Only Orders

**File:** [08_post_only_orders/main.go](08_post_only_orders/main.go)

A post-only order guarantees **maker** status.  If placing it would result in
an immediate match (taker execution), it is rejected rather than executed.
Exchanges use this to let market-makers earn the maker rebate.

Key ideas:
- `NewPostOnlyOrder(id, side, qty, price, tif, oco)`
- Crossing post-only: `done.Stored=false`, zero trades
- Non-crossing post-only: rests normally, `done.Stored=true`

---

## 09 · Snapshot (Dump & Restore)

**File:** [09_snapshot/main.go](09_snapshot/main.go)

The full engine state (bids, asks, stop orders, OCO pairs) can be serialized
to a plain `Snapshot` struct and restored into a fresh engine later.

Useful for persistence, hot-standby replication, and deterministic test setup.

Key ideas:
- `ob.Dump() Snapshot` — returns a serialization-friendly struct
- `ob.Restore(snap) error` — clears the engine and rebuilds from snapshot
- `Snapshot` contains only exported plain-data fields; marshal with `encoding/json`

---

## 10 · Event Emitter

**File:** [10_event_emitter/main.go](10_event_emitter/main.go)

Attach an `EventEmitter` to receive real-time notifications without polling.
Use this to push updates over WebSocket, publish to a message bus, or build
an audit log.

| Event | Trigger |
|-------|---------|
| `EventTrade` | A fill occurred |
| `EventVolume` | Volume at a price level changed |
| `EventDepth` | `ob.Depth()` was called |
| `EventCanceled` | An order was canceled |
| `EventActivated` | A stop-limit order became live |

Key ideas:
- Implement `matchingo.EventEmitter` (single `Emit` method)
- `ob.SetEmitter(e)` / `ob.SetEmitter(nil)` to enable or disable
- Use a type-switch on `data any` for typed event payloads

---

## 11 · Precision Limits

**File:** [11_precision_limits/main.go](11_precision_limits/main.go)

Enforce maximum decimal places for quantities and prices on a per-book basis.
Orders that exceed the configured precision are rejected by `Process` with a typed error.

Key ideas:
- `ob.SetQuantityPrecision(n)` — reject quantities with more than `n` decimal places
- `ob.SetPricePrecision(n)` — reject prices (and stop prices) with more than `n` decimal places
- Default `0` means no limit
- Returns `ErrQuantityPrecision` or `ErrPricePrecision` — no panic, no silent truncation

---

## Running all examples at once

```shell
for d in examples/*/; do
  echo "=== $d ===" && go run ./$d
done
```
