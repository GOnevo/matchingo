# TODO — features present in common matching engines but missing here

## Order Types

- **Stop-Market order** — current stop orders always activate as limit; a stop-market variant (no price, matches at best available) is a standard primitive
- **Trailing stop** — stop price tracks the best price by a fixed offset or percentage; common in retail and prop-trading engines
- **Market-to-Limit (MTL)** — a market order that, if partially filled, converts the remainder into a limit order at the last trade price
- **Day / GTD (Good-Till-Date)** — TIF that auto-cancels at a configurable wall-clock time; only GTC / IOC / FOK exist today

## Matching Logic

- **Price improvement / mid-price matching** — some venues match at the mid-point between bid and ask instead of the aggressor's price; relevant for dark pools and internalisation engines
- **Self-trade prevention (STP)** — cancel or decrement when both sides of a potential trade share the same participant ID; required by most exchange regulations and absent here
- **Minimum quantity / Min-fill (MFQ)** — an order that must fill at least N units on the first match attempt or be cancelled entirely; common on FX and equities markets
- **Maximum show size on iceberg** — currently peak refills unconditionally; some engines randomise the refill size within a range to reduce detectability

## Order Lifecycle

- **Modify stop price** — `Modify` only changes price/qty on resting limit orders; stop-limit orders cannot have their stop price amended
- **Partial cancel (reduce-only)** — explicit API to reduce quantity without full cancel; some engines treat a quantity-decrease modify differently from a cancel+resubmit to preserve audit trail
- **Order expiry / time-to-live** — engine-side TTL with automatic cancellation; currently all lifecycle management is external
- **Replace (cancel-replace atomic)** — a single atomic swap that guarantees no fill between the cancel and the new order; different from the current Modify re-queue which goes through Process

## Sequence & Audit

- **Sequence number on Done / events** — `Snapshot.Sequence` exists but is never incremented; trades and events carry no monotonic sequence, making replay and gap detection impossible
- **Order timestamps** — no creation or last-update timestamp on `Order`; required for GTD expiry, audit logs, and latency analytics
- **Trade ID** — individual trades have no unique identifier; standard for FIX/ITCH/PITCH protocols and clearing systems

## Market Data

- **Full depth snapshot with sequence** — `Dump()` captures state but `Snapshot.Sequence` is always 0; a sequenced snapshot paired with an incremental feed (like ITCH AddOrder / DeleteOrder messages) is the standard dissemination model
- **Top-of-book (BBO) change event** — no dedicated event when the best bid or offer changes; callers must infer it from volume events
- **Trade print / last-sale event** — `EventTrade` exists but fires twice (once per side) with inconsistent price field (taker side uses `done.Order.Price()` not the actual trade price); a single canonical trade-print event per fill is standard

## Infrastructure

- **Concurrency / thread safety** — the order book has no internal locking; safe single-threaded use only; production engines either use a dedicated goroutine with a channel-based API or fine-grained locks
- **Persistence / WAL** — `Dump`/`Restore` provide point-in-time snapshots but there is no write-ahead log; crash recovery requires replaying all orders since the last snapshot
- **Instrument / symbol metadata** — no tick-size, lot-size, or price-band validation; real engines reject orders that violate instrument constraints before they reach the book
- **Circuit breakers / trading halts** — no mechanism to pause matching or reject new orders while keeping the book intact
