# matchingo — Order Book Matching Engine

## 1. Purpose

matchingo is a single-instrument **limit order book** and **matching engine**. It accepts orders, matches them against resting liquidity by **price-time priority**, keeps the remainder on the book when allowed, and reports every outcome (trades, cancellations, activations) both as a return value and as an event stream.

It is deterministic: given the same configuration, the same starting state and the same ordered sequence of commands (with the same clock readings), it always produces the same book, the same trades and the same IDs. This property underlies crash recovery (WAL replay) and snapshotting.

All prices and quantities are **fixed-point decimals** (no binary floating point), so arithmetic is exact and comparisons are reliable.

## 2. Core Concepts

| Concept | Meaning |
|---|---|
| **Side** | BUY (bids) or SELL (asks). |
| **Bid side** | Resting buy orders. Best price = highest. |
| **Ask side** | Resting sell orders. Best price = lowest. |
| **Price level** | All resting orders at one price on one side, kept as a FIFO queue. |
| **Maker** | The resting order that provides liquidity. |
| **Taker** | The incoming order that consumes liquidity. |
| **Stop book** | A separate store of dormant stop orders, keyed by stop price. They are not part of visible liquidity and cannot trade until activated. |
| **Quote quantity** | A quantity expressed in money (price × amount) instead of units. Only used by market-quote orders. |
| **Sequence** | A counter that increases by one for every accepted new order (including the new order of a replace). |
| **Trade ID** | A counter that increases by one for every individual fill. |

### Invariants

- A price level exists only while it contains at least one order.
- Orders at the same price are served strictly in arrival order (time priority).
- A trade always executes at the **maker's price**, never the taker's limit.
- The best bid is always strictly lower than the best ask after any command completes (crossing orders match immediately; post-only crossing orders are rejected from resting).
- An order ID is unique across resting and dormant (stop) orders.
- Dormant stop orders never match and never appear in depth, best bid/ask or level volume.

## 3. Matching Algorithm

### 3.1 Priority

1. **Price priority**: the buy side matches the lowest ask first; the sell side matches the highest bid first.
2. **Time priority**: within a price level, the oldest resting order is filled first.

### 3.2 Generic matching loop

For an incoming order with remaining quantity *Q*:

1. Look at the best opposite price level.
2. Stop if *Q* = 0, the opposite side is empty, or (for limit orders) the best opposite price no longer satisfies the limit: a BUY limit crosses while `ask ≤ limit`, a SELL limit crosses while `bid ≥ limit`. Market orders have no price limit.
3. Walk the level's queue from the oldest order:
   - If *Q* < maker quantity: the maker is **partially filled** (keeps its queue position, quantity reduced), *Q* becomes 0.
   - Otherwise the maker is **fully filled** and removed; *Q* decreases by the maker's quantity.
   - Each fill creates one trade at the level price and increments the trade ID.
4. Move to the next-best level and repeat.

After the loop the fate of the remainder depends on order type and time-in-force (§4, §5).

### 3.3 Result report

Every accepted order produces a result containing:

- **Order**: the incoming order.
- **Trades**: the first entry is a **summary of the taker** (total matched quantity); the following entries are one per maker fill (maker ID, price, quantity). Empty if nothing traded.
- **Processed** (matched amount), **Left** (unmatched amount), **Quantity** (original amount for this run).
- **Stored**: whether the remainder now rests on the book.
- **Canceled**: IDs of every order canceled as a side effect (the incoming order itself, STP victims, OCO partners, expired orders).
- **Activated / Activations**: stop orders triggered by this order's trades, and the nested results of processing each of them.
- **Sequence**: the sequence number assigned to the command.

### 3.4 Side effects of a trade

Whenever at least one fill happened at a price level, the following run in this order:

1. **Circuit breaker check** (§9.2), if enabled.
2. **Trailing stop adjustment** (§4.6) using that level price as the trade price.
3. **Stop activation**: stops whose trigger is met by the level price are collected (§4.5). They are processed only after the current order has finished its own matching (§4.5).

## 4. Order Types

### 4.1 Limit order

Has a price and quantity. Matches while the opposite side crosses its price; any remainder **rests** at its limit price at the back of that level's queue (subject to TIF).

- If nothing matched at all, the whole order rests.
- If partly matched, the remainder rests with the reduced quantity.
- If fully matched, it never rests.

### 4.2 Market order

Has a side and quantity but no price. Takes liquidity at any price until filled or the opposite side is empty. **Never rests**: any unfilled remainder is canceled and reported in `Canceled`.

### 4.3 Market quote order

A market order whose quantity is a **money budget**. At each price level the engine converts the remaining budget to units (`budget ÷ level price`), fills what it can, and subtracts the money actually spent (`filled units × level price`). Matching stops when the budget is exhausted or cannot buy any unit at the best price (dust). The unspent remainder is canceled. Quote orders are exempt from lot-size checks since the quantity is not a number of units.

### 4.4 Market-to-limit order

A market order that, after consuming liquidity, **rests its remainder as a limit order priced at the last trade price** (the price of its final fill) instead of being canceled. If it made no fill at all, or was stopped by self-trade prevention, it is canceled like a plain market order.

### 4.5 Stop orders (stop-limit and stop-market)

A stop order is dormant until the market trades through its **stop price**; then it becomes a live order:

- **Stop-limit** → becomes a limit order at its limit price.
- **Stop-market** → becomes a market order.

**Trigger rules** (evaluated against the price of each traded price level):

- BUY stop triggers when `stop price ≤ trade price` (price rose to the stop).
- SELL stop triggers when `stop price ≥ trade price` (price fell to the stop).

Important properties:

- Triggering is driven **only by trades**. Submitting a stop order never triggers it immediately, even if the last trade price is already beyond the stop; it waits for the next qualifying trade.
- Stops are collected buy-side first then sell-side. Within a side they activate in order of stop price (BUY: lowest first, SELL: highest first), and in arrival order within the same stop price.
- Activated orders are **processed after** the order that caused the trade finishes, one by one. Their own trades can in turn trigger further stops (cascade). Each activation is reported nested in `Activations`.
- An activated stop-limit that finds no liquidity rests like any limit order; an activated stop-market that finds none is canceled.
- A stop-limit may carry a time-in-force that applies once it activates.
- Dormant stops can be canceled, modified (§6.4), expired (GTD) or linked via OCO.

### 4.6 Trailing stop order

A stop-market order whose stop price **follows the market at a fixed distance**, in absolute units or as a percentage of the trade price (strictly between 0 and 100).

On every trade at price *P*, each dormant trailing stop recomputes a candidate:

- SELL: `candidate = P − offset`; the stop moves **up** only if the candidate is higher than the current stop.
- BUY: `candidate = P + offset`; the stop moves **down** only if the candidate is lower than the current stop.

For a percentage trail, `offset = P × percent / 100`. The stop never moves in the unfavorable direction, so it ratchets protectively. A moved stop is re-queued at the back of its new stop-price level. Adjustment happens before the activation check of the same trade, and trailing stops are processed in creation order so the result is deterministic. Once activated, it behaves as a plain market order.

### 4.7 Iceberg order

A limit order showing only a **peak** (visible) quantity while the rest of the **total** is hidden. The peak must be smaller than the total.

- The book (depth, volume, queue) only sees the visible slice.
- When the visible slice is completely filled, the next slice (the smaller of peak and remaining hidden) is **re-posted at the back of the queue**, so each refill loses time priority.
- For FOK checks the whole total (visible + hidden) must be fillable.

### 4.8 Post-only order

A limit order that must **add** liquidity. If, at arrival, it would immediately match (the best opposite price crosses its limit), it is **canceled, not rejected and never traded**. Otherwise it rests normally.

### 4.9 Good-till-date (GTD) order

A limit or stop order that carries an **expiry timestamp** (or a TTL, a duration added to the book clock at the moment of acceptance). It is canceled once the clock reaches its expiry (`expiry ≤ now`). An order whose expiry is already in the past is rejected. TTL works only with GTC/unset time-in-force and never with market orders.

See §8 for how expiry is enforced.

## 5. Time-in-Force

| TIF | Behavior |
|---|---|
| **GTC** (default) | Remainder rests until filled or canceled. |
| **IOC** | Match what is possible immediately; the unmatched remainder is canceled and never rests. |
| **FOK** | Before matching, check that the **entire** quantity can be filled at prices within the limit. If not, the order is canceled with **no trades at all**. If yes, it fills completely. |
| **GTD** | Rests until its expiry time (§4.9). |

The FOK feasibility check sums resting volume from the best level outward while levels still satisfy the limit price. FOK cannot be combined with self-trade prevention (rejected as invalid), because STP could make a pre-checked fill impossible.

## 6. Order Management Operations

### 6.1 Submit

Validation, in order: trading not halted; order well-formed (known type, valid side, not already canceled); quantity > 0; quantity/price within decimal precision; instrument rules (§9.3); FOK/STP compatibility; TTL compatibility; expiry not in the past; ID not already in use. A rejected order changes nothing and is not logged to the WAL.

Accepted orders: expired orders are purged if the clock passed the earliest expiry (§8); a creation timestamp is set; the sequence number increments; then the order is dispatched to market / limit / stop handling.

### 6.2 Cancel

Removes a resting or dormant order by ID, marks it canceled and emits a cancel event. Unknown ID → nothing happens. Cancel works even while trading is halted.

### 6.3 Modify a resting limit order

- **Same price and new quantity ≤ current** (quantity decrease): done **in place**; the order keeps its time priority. No new matching occurs.
- **Any other change** (price change or quantity increase): the order loses priority; it is removed and re-submitted as a fresh limit order at the back of the new level and can match immediately if it now crosses.

Not allowed for stop orders, unknown orders or while halted. New values must satisfy instrument rules.

### 6.4 Modify a dormant stop order

Changes stop price, limit price (ignored for stop-market) and quantity. The order is removed and re-inserted at the back of its new stop-price level (priority lost).

### 6.5 Replace

An **atomic cancel-and-submit**: the new order is fully validated *before* anything is touched; if validation fails the old order stays untouched. The new order may reuse the old ID. It goes to the back of its queue and matches as a normal new order. The old ID is listed first in the result's `Canceled`.

### 6.6 Queries

- Look up an order by ID.
- Best bid / best ask (price and total volume at that level; empty if none).
- Volume at a specific price on a side.
- **Market price estimate**: total cost (buying) or proceeds (selling) of consuming a given quantity from the opposite side level by level; fails if the book lacks enough volume. Does not change the book.
- **Depth**: all price levels with aggregate volume per side.

## 7. Linked and Protective Mechanisms

### 7.1 OCO (one-cancels-the-other)

An order may name another order ID as its OCO partner. When an order **executes to completion** (a taker that is fully matched, a market order that traded, or a resting maker that is completely filled), its partner is canceled.

- A **partial fill alone does not trigger** OCO.
- If the partner is not on the book yet, the trigger is remembered; when the partner later arrives it is canceled immediately instead of being processed.
- Partners may be limit, stop or dormant orders.

### 7.2 Self-trade prevention (STP)

An order may be tagged with a participant ID and an STP mode. When the incoming (taker) order would trade against a resting order of the **same participant**, the taker's mode decides:

| Mode | Effect |
|---|---|
| None | Self-trade is allowed. |
| **Cancel newest** | The incoming order's remainder stops matching and is canceled; the resting order stays. |
| **Cancel oldest** | The resting order is canceled; the incoming order continues matching further orders. |
| **Cancel both** | The resting order is canceled and the incoming remainder stops and is canceled. |
| **Decrement** | Both are reduced by the overlapping amount **without producing a trade**. If the resting order is fully consumed it is canceled; if the incoming order is fully consumed it ends. |

Trades earlier in the same run remain valid. Decrement on a market-quote order falls back to cancel-newest (a money budget cannot be reduced in units).

## 8. Expiry Enforcement (GTD)

The book tracks a lower bound of the earliest expiry time. Expiry is enforced:

- **Lazily**: before every new order is processed, if the clock has reached the earliest expiry, all due orders are canceled (in expiry order, ties by ID). Their IDs appear in the new order's `Canceled`.
- **Explicitly**: an expire operation can be invoked at any time (e.g., from a timer) to cancel everything due at a given instant on an otherwise idle book.

The time source is injectable so tests and replays are fully deterministic.

## 9. Market Controls

### 9.1 Trading halt

A manual halt makes the book **reject all new orders and amendments** (submit, replace, modify). Cancels and expiry still work, and the book contents stay intact. Resume re-opens trading. State changes are announced as events.

### 9.2 Circuit breaker

Optionally configured with a percentage. After trading at a price level, the engine compares it to the **previous trade price**; if the absolute move exceeds that percentage the book **halts automatically**.

- The order that caused the breach finishes normally (including its remaining matches and stop activations); only later orders are rejected.
- The last trade price is retained across halts and resume, and is part of saved state.
- The first trade of a book (no reference yet) never trips the breaker.

### 9.3 Instrument constraints

An optional instrument definition validates every order and modification:

- **Tick size**: prices (limit and stop) must be multiples.
- **Lot size**: quantities must be multiples (for icebergs both the visible and hidden slices).
- **Min / max quantity**: bounds on total quantity (total for icebergs).
- **Min / max price**: price band for limit and stop prices.

A zero field means "unchecked". Violations are rejected before touching the book. An inconsistent definition (negative or min > max) is refused at configuration.

### 9.4 Precision limits

Separate maximum decimal places can be set for quantity and price. Values with more decimal places are rejected.

## 10. Persistence and Recovery

### 10.1 Snapshot

A complete, plain-data image of the engine: sequence, trade ID, last WAL position, halt flag, last trade price, every resting bid/ask in queue order, every dormant stop, and the pending OCO markers. Restoring clears the current state and loads the image (and recomputes the next expiry). Snapshots carry a version; unknown versions are refused. Order attributes needed to continue behaving identically (iceberg hidden remainder, trail settings, expiry, STP tag, creation time) are preserved.

### 10.2 Write-ahead log (WAL)

When a log is attached, every **state-changing command** (submit, replace, cancel, modify, modify-stop, expire, halt, resume) is written **before** it is applied, with a log sequence number and the clock reading at that moment. Rules:

- Commands rejected by validation are not logged.
- The book clock is **frozen to the logged time** while the command runs so live execution equals replay.
- If the log write fails, the command is **not applied**, the error is latched and all further mutations are refused until the log is reattached/reset.
- The built-in format is one JSON record per line, flushed to durable storage per record when the sink supports it; a torn last line (crash mid-write, command never applied) is ignored on read.

### 10.3 Recovery

State = latest snapshot + replay of log records after the snapshot's log position. Replay requires **consecutive** sequence numbers (gaps → error), feeds each record's time as the clock, re-applies commands in order, and does not write to the log. Commands that were rejected originally are rejected again, so replay reproduces exactly the original state. The book must be configured the same way (precision, instrument, breaker) as when the log was written.

## 11. Events

An optional emitter receives events as the book changes (sent synchronously, stamped with the current sequence):

| Event | When |
|---|---|
| **trade** | Every fill: trade ID, maker ID, taker ID, price, amount, sequence. |
| **volume** | Aggregate volume at a price level changed (order added, filled, removed, reduced, STP decrement, iceberg refill). |
| **canceled** | An order was canceled for any reason. |
| **activated** | A stop order triggered. |
| **bbo** | Best bid or best ask changed after a command. |
| **depth** | Emitted whenever a depth snapshot is requested. |
| **halt** | Trading halted or resumed (flag whether the circuit breaker caused it). |

Delivery can be a channel with either backpressure (blocks when full) or drop-on-full.

## 12. Concurrency Model

The core engine is single-threaded by design (no locking overhead). A wrapper makes it safe for concurrent callers: every command takes an exclusive lock so each executes atomically and the overall outcome equals some sequential order (the WAL records exactly that order); pure reads share a read lock. Event emitters and the WAL run under the lock and must not call back into the book. Lookups return a copy of an order; objects passed in by the caller should not be read after submission except through lookup.

## 13. Errors

Rejections are explicit and leave the book unchanged:

- invalid order / side / quantity / price / time-in-force; already-canceled order
- duplicate order ID; order not found; wrong order kind for modify (limit vs stop)
- quantity or price precision exceeded
- instrument violations: tick, lot, quantity range, price band; invalid instrument definition
- order already expired; invalid STP setting
- trading halted
- insufficient volume for a price estimate
- WAL failure, WAL gap, invalid WAL record

## 14. Data Structures and Complexity (logical view)

- **Each side**: price → FIFO queue map plus a sorted list of active prices (best first). Insert/remove of a price level is a binary search plus array shift; best-price lookup is O(1); next-level is O(log n).
- **Price level queue**: constant-time append, pop-front and (indexed) removal, with a cached aggregate volume so level volume is O(1).
- **Order index**: ID → order for O(1) lookup, cancel and duplicate detection.
- **Stop book**: stop price → FIFO queue plus ID index. Activation scans the stop price levels; trailing adjustment scans dormant trailing stops.
- **FOK check**: walks levels from the best price until the quantity is covered or the limit is breached.
- **Expiry**: purge scans resting orders, but only when the clock has passed the earliest known expiry.
