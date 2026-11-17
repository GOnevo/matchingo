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

- **Partial cancel (reduce-only)** — explicit API to reduce quantity without full cancel; some engines treat a quantity-decrease modify differently from a cancel+resubmit to preserve audit trail
- **Order expiry / time-to-live** — engine-side TTL with automatic cancellation; currently all lifecycle management is external
- **Replace (cancel-replace atomic)** — a single atomic swap that guarantees no fill between the cancel and the new order; different from the current Modify re-queue which goes through Process

## Infrastructure

- **Concurrency / thread safety** — the order book has no internal locking; safe single-threaded use only; production engines either use a dedicated goroutine with a channel-based API or fine-grained locks
- **Persistence / WAL** — `Dump`/`Restore` provide point-in-time snapshots but there is no write-ahead log; crash recovery requires replaying all orders since the last snapshot
- **Instrument / symbol metadata** — no tick-size, lot-size, or price-band validation; real engines reject orders that violate instrument constraints before they reach the book
- **Circuit breakers / trading halts** — no mechanism to pause matching or reject new orders while keeping the book intact
