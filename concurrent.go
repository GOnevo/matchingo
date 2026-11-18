package matchingo

import (
	"sync"
	"time"
)

// ConcurrentOrderBook makes an OrderBook safe for use from multiple goroutines.
//
// Commands are serialized by one lock, so every call takes effect atomically and the
// outcome equals some single-threaded order (with a WAL attached, the log is exactly
// that order). Pure reads share a read lock. OrderBook itself stays lock-free, so
// single-threaded callers pay nothing.
//
// Things to know:
//   - Event emitters and the WAL run while the lock is held. They must not call back
//     into the book (deadlock), and a slow one slows every caller.
//   - Pointers handed out are not guarded. GetOrder returns a copy, but the *Order you
//     passed to Process, Done.Order and Done.Activations still point at live book state:
//     after Process returns, read your order through GetOrder, not through the pointer.
//   - For anything not wrapped here, use View (read) or Update (write).
type ConcurrentOrderBook struct {
	mu sync.RWMutex
	ob *OrderBook
}

// NewConcurrentOrderBook creates a goroutine-safe order book with the given options.
func NewConcurrentOrderBook(opts ...Option) *ConcurrentOrderBook {
	return &ConcurrentOrderBook{ob: NewOrderBook(opts...)}
}

// View runs fn with shared read access to the underlying book. fn must not modify the
// book or keep references to its internals after returning.
func (c *ConcurrentOrderBook) View(fn func(ob *OrderBook)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fn(c.ob)
}

// Update runs fn with exclusive access to the underlying book.
func (c *ConcurrentOrderBook) Update(fn func(ob *OrderBook)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(c.ob)
}

// Process see OrderBook.Process.
func (c *ConcurrentOrderBook) Process(order *Order) (*Done, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.Process(order)
}

// Replace see OrderBook.Replace.
func (c *ConcurrentOrderBook) Replace(oldID string, newOrder *Order) (*Done, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.Replace(oldID, newOrder)
}

// CancelOrder see OrderBook.CancelOrder.
func (c *ConcurrentOrderBook) CancelOrder(orderID string) *Order {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.CancelOrder(orderID)
}

// Modify see OrderBook.Modify.
func (c *ConcurrentOrderBook) Modify(orderID string, newPrice, newQty Decimal) (*Done, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.Modify(orderID, newPrice, newQty)
}

// ModifyStop see OrderBook.ModifyStop.
func (c *ConcurrentOrderBook) ModifyStop(orderID string, newStopPrice, newLimitPrice, newQty Decimal) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.ModifyStop(orderID, newStopPrice, newLimitPrice, newQty)
}

// ExpireOrders see OrderBook.ExpireOrders.
func (c *ConcurrentOrderBook) ExpireOrders(now time.Time) []*Order {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.ExpireOrders(now)
}

// Halt see OrderBook.Halt.
func (c *ConcurrentOrderBook) Halt() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.Halt()
}

// Resume see OrderBook.Resume.
func (c *ConcurrentOrderBook) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.Resume()
}

// Restore see OrderBook.Restore.
func (c *ConcurrentOrderBook) Restore(snap Snapshot) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.Restore(snap)
}

// Replay see OrderBook.Replay.
func (c *ConcurrentOrderBook) Replay(records []WALRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.Replay(records)
}

// SetWAL see OrderBook.SetWAL.
func (c *ConcurrentOrderBook) SetWAL(w WAL) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.SetWAL(w)
}

// SetEmitter see OrderBook.SetEmitter.
func (c *ConcurrentOrderBook) SetEmitter(e EventEmitter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.SetEmitter(e)
}

// SetEventChannel see OrderBook.SetEventChannel.
func (c *ConcurrentOrderBook) SetEventChannel(ch chan<- Event, dropOnFull bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.SetEventChannel(ch, dropOnFull)
}

// SetQuantityPrecision see OrderBook.SetQuantityPrecision.
func (c *ConcurrentOrderBook) SetQuantityPrecision(decimals int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.SetQuantityPrecision(decimals)
}

// SetPricePrecision see OrderBook.SetPricePrecision.
func (c *ConcurrentOrderBook) SetPricePrecision(decimals int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.SetPricePrecision(decimals)
}

// SetCircuitBreaker see OrderBook.SetCircuitBreaker.
func (c *ConcurrentOrderBook) SetCircuitBreaker(percent Decimal) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ob.SetCircuitBreaker(percent)
}

// Depth see OrderBook.Depth. It emits EventDepth, so it takes the write lock.
func (c *ConcurrentOrderBook) Depth() *Depth {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.Depth()
}

// DepthJSON see OrderBook.DepthJSON.
func (c *ConcurrentOrderBook) DepthJSON() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ob.DepthJSON()
}

// GetOrder returns a copy of the order with the given ID, or nil. The copy does not
// change as the book matches.
func (c *ConcurrentOrderBook) GetOrder(orderID string) *Order {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o := c.ob.GetOrder(orderID)
	if o == nil {
		return nil
	}
	cp := *o
	if o.ext != nil {
		ext := *o.ext
		cp.ext = &ext
	}
	return &cp
}

// CalculateMarketPrice see OrderBook.CalculateMarketPrice.
func (c *ConcurrentOrderBook) CalculateMarketPrice(side Side, quantity Decimal) (Decimal, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.CalculateMarketPrice(side, quantity)
}

// BestBid see OrderBook.BestBid.
func (c *ConcurrentOrderBook) BestBid() *Level {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.BestBid()
}

// BestAsk see OrderBook.BestAsk.
func (c *ConcurrentOrderBook) BestAsk() *Level {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.BestAsk()
}

// Level see OrderBook.Level.
func (c *ConcurrentOrderBook) Level(side Side, price Decimal) (Decimal, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.Level(side, price)
}

// Dump see OrderBook.Dump.
func (c *ConcurrentOrderBook) Dump() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.Dump()
}

// Sequence see OrderBook.Sequence.
func (c *ConcurrentOrderBook) Sequence() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.Sequence()
}

// WALSeq see OrderBook.WALSeq.
func (c *ConcurrentOrderBook) WALSeq() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.WALSeq()
}

// WALError see OrderBook.WALError.
func (c *ConcurrentOrderBook) WALError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.WALError()
}

// IsHalted see OrderBook.IsHalted.
func (c *ConcurrentOrderBook) IsHalted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.IsHalted()
}

// Instrument returns a copy of the configured instrument, or nil.
func (c *ConcurrentOrderBook) Instrument() *Instrument {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ob.instrument == nil {
		return nil
	}
	in := *c.ob.instrument
	return &in
}

// String implements fmt.Stringer.
func (c *ConcurrentOrderBook) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ob.String()
}
