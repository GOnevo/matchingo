package matchingo

import (
	"sort"
	"strings"
	"time"

	"github.com/nikolaydubina/fpdecimal"
)

// OrderBook implements standard matching algorithm
type OrderBook struct {
	orders            map[string]*Order
	asks              *OrderSide
	bids              *OrderSide
	Stop              *StopBook
	OCO               map[string]struct{}
	emitter           EventEmitter
	sequence          uint64
	tradeID           uint64
	instrument        *Instrument
	wal               WAL
	walSeq            uint64 // sequence of the last record written to (or replayed from) the WAL
	walErr            error
	halted            bool
	lastPrice         Decimal // last trade price level, the circuit breaker reference
	breakerPct        Decimal // halt when a trade moves more than this percent from lastPrice; 0 = off
	clock             func() time.Time
	nextExpiry        time.Time // lower bound of the earliest GTD expiry; zero = none
	quantityPrecision int       // max decimal places for quantity; 0 = unlimited
	pricePrecision    int       // max decimal places for price; 0 = unlimited
}

// Sequence returns the current monotonic sequence counter.
// It is incremented once per successful Process call.
func (ob *OrderBook) Sequence() uint64 { return ob.sequence }

// NewOrderBook creates Orderbook object
// Option configures an OrderBook at construction time.
type Option func(*OrderBook)

// WithQuantityPrecision sets the maximum decimal places allowed for quantities.
func WithQuantityPrecision(decimals int) Option {
	return func(ob *OrderBook) { ob.quantityPrecision = decimals }
}

// WithPricePrecision sets the maximum decimal places allowed for prices.
func WithPricePrecision(decimals int) Option {
	return func(ob *OrderBook) { ob.pricePrecision = decimals }
}

// WithEmitter sets the event emitter at construction time.
func WithEmitter(e EventEmitter) Option {
	return func(ob *OrderBook) { ob.emitter = e }
}

// WithClock sets the time source used for GTD expiry, order timestamps and WAL records (default: current UTC time).
func WithClock(clock func() time.Time) Option {
	return func(ob *OrderBook) { ob.clock = clock }
}

// WithInstrument validates every order against the instrument's tick size, lot size,
// quantity range and price band. It panics on a negative or inverted definition.
func WithInstrument(in Instrument) Option {
	in.validate()
	return func(ob *OrderBook) { ob.instrument = &in }
}

// Instrument returns the configured instrument, or nil if none.
func (ob *OrderBook) Instrument() *Instrument { return ob.instrument }

// WithCircuitBreaker halts trading when a trade price moves more than percent
// (5 means 5%) away from the previous trade price. 0 disables it.
func WithCircuitBreaker(percent Decimal) Option {
	return func(ob *OrderBook) { ob.breakerPct = percent }
}

// WithPrecision sets both quantity and price precision in one call.
func WithPrecision(quantityDecimals, priceDecimals int) Option {
	return func(ob *OrderBook) {
		ob.quantityPrecision = quantityDecimals
		ob.pricePrecision = priceDecimals
	}
}

// utcNow is the default clock: UTC without the monotonic reading, so times survive
// a JSON round trip (WAL, snapshots) unchanged.
func utcNow() time.Time { return time.Now().UTC().Round(0) }

func NewOrderBook(opts ...Option) *OrderBook {
	ob := &OrderBook{
		orders: map[string]*Order{},
		bids:   NewOrderSideBid(),
		asks:   NewOrderSideAsk(),
		Stop:   NewStopBook(),
		OCO:    map[string]struct{}{},
		clock:  utcNow,
	}
	for _, o := range opts {
		o(ob)
	}
	return ob
}

// SetQuantityPrecision sets the maximum number of decimal places allowed for order quantities.
// A value of 0 disables the check.
func (ob *OrderBook) SetQuantityPrecision(decimals int) {
	ob.quantityPrecision = decimals
}

// SetPricePrecision sets the maximum number of decimal places allowed for order prices.
// A value of 0 disables the check.
func (ob *OrderBook) SetPricePrecision(decimals int) {
	ob.pricePrecision = decimals
}

// checkPrecision returns an error if d has more decimal places than allowed.
func checkPrecision(d Decimal, maxDecimals int) bool {
	if maxDecimals == 0 {
		return true
	}
	fraction := int(fpdecimal.FractionDigits)
	if maxDecimals >= fraction {
		return true
	}
	factor := int64(1)
	for i := 0; i < fraction-maxDecimals; i++ {
		factor *= 10
	}
	return d.Scaled()%factor == 0
}

// Halt stops the book accepting new orders and amendments (Process, Replace, Modify,
// ModifyStop return ErrTradingHalted). Cancels and expiry keep working and the book stays intact.
func (ob *OrderBook) Halt() {
	if ob.wal != nil {
		ob.haltLogged(WALHalt, true)
		return
	}
	ob.setHalted(true, false)
}

// Resume re-opens a halted book. The last trade price stays the circuit breaker reference.
func (ob *OrderBook) Resume() {
	if ob.wal != nil {
		ob.haltLogged(WALResume, false)
		return
	}
	ob.setHalted(false, false)
}

// IsHalted reports whether trading is halted.
func (ob *OrderBook) IsHalted() bool { return ob.halted }

// SetCircuitBreaker sets the price-move circuit breaker threshold in percent; 0 disables it.
func (ob *OrderBook) SetCircuitBreaker(percent Decimal) { ob.breakerPct = percent }

func (ob *OrderBook) setHalted(halted, auto bool) {
	if ob.halted == halted {
		return
	}
	ob.halted = halted
	ob.emit(EventHalt, HaltEventData{Halted: halted, Auto: auto})
}

// checkBreaker records a trade price level and halts when it moved too far from the previous one.
// The order being processed still completes; only later orders are rejected.
func (ob *OrderBook) checkBreaker(price Decimal) {
	last := ob.lastPrice
	ob.lastPrice = price
	if last.LessThanOrEqual(Zero) {
		return
	}
	move := price.Sub(last)
	if move.LessThan(Zero) {
		move = Zero.Sub(move)
	}
	if move.Mul(FromInt(100)).GreaterThan(last.Mul(ob.breakerPct)) {
		ob.setHalted(true, true)
	}
}

// GetOrder returns Order by id
func (ob *OrderBook) GetOrder(orderID string) *Order {
	order, ok := ob.orders[orderID]
	if !ok {
		return nil
	}

	return order
}

// CancelOrder removes Order with given ID from the Order book or the Stop book
func (ob *OrderBook) CancelOrder(orderID string) *Order {
	if ob.wal != nil {
		return ob.cancelOrderLogged(orderID)
	}
	return ob.cancelOrder(orderID)
}

// cancelOrder is CancelOrder without the WAL gate.
func (ob *OrderBook) cancelOrder(orderID string) *Order {
	order := ob.GetOrder(orderID)
	if order == nil {
		return nil
	}

	var prevBid, prevAsk Decimal
	if ob.emitter != nil {
		prevBid, prevAsk = ob.bboSnapshot()
	}

	order.Cancel()

	if order.IsStopOrder() {
		ob.Stop.Remove(order)
		delete(ob.orders, order.ID())
	} else {
		ob.deleteOrder(order)
	}

	ob.emit(EventCanceled, orderID)
	ob.emitBBOIfChanged(prevBid, prevAsk)
	return order
}

// ExpireOrders cancels every resting GTD order whose expiry is at or before now
// and returns them in expiry order. Process also calls it automatically when the
// book clock passes the earliest expiry; call it from a timer to expire orders
// on an otherwise idle book. It scans all resting orders.
func (ob *OrderBook) ExpireOrders(now time.Time) []*Order {
	if ob.wal != nil {
		return ob.expireOrdersLogged(now)
	}
	return ob.expireOrders(now)
}

// expireOrders is ExpireOrders without the WAL gate.
func (ob *OrderBook) expireOrders(now time.Time) []*Order {
	var expired []*Order
	for _, o := range ob.orders {
		if o.tif == GTD && !o.extra().expiresAt.After(now) {
			expired = append(expired, o)
		}
	}
	sort.Slice(expired, func(i, j int) bool {
		if !expired[i].extra().expiresAt.Equal(expired[j].extra().expiresAt) {
			return expired[i].extra().expiresAt.Before(expired[j].extra().expiresAt)
		}
		return expired[i].id < expired[j].id
	})
	for _, o := range expired {
		ob.cancelOrder(o.ID())
	}
	ob.refreshNextExpiry()
	return expired
}

// refreshNextExpiry recomputes nextExpiry from the resting GTD orders.
func (ob *OrderBook) refreshNextExpiry() {
	ob.nextExpiry = time.Time{}
	for _, o := range ob.orders {
		if o.tif == GTD && (ob.nextExpiry.IsZero() || o.extra().expiresAt.Before(ob.nextExpiry)) {
			ob.nextExpiry = o.extra().expiresAt
		}
	}
}

// Modify changes the price or quantity of a resting limit order.
// Returns false if the order is not found or is a stop-limit order.
// In-place decrease: same price and newQty ≤ current open → keep time priority.
// Re-queue: price change or quantity increase → remove and re-insert at back of new price level, then match.
func (ob *OrderBook) Modify(orderID string, newPrice, newQty Decimal) (*Done, error) {
	if ob.wal != nil {
		return ob.modifyLogged(orderID, newPrice, newQty)
	}
	return ob.modify(orderID, newPrice, newQty)
}

// modify is Modify without the WAL gate.
func (ob *OrderBook) modify(orderID string, newPrice, newQty Decimal) (*Done, error) {
	if ob.halted {
		return nil, ErrTradingHalted
	}
	order := ob.GetOrder(orderID)
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.IsStopOrder() {
		return nil, ErrNotLimitOrder
	}
	if ob.instrument != nil {
		if err := ob.instrument.checkPrice(newPrice); err != nil {
			return nil, err
		}
		if err := ob.instrument.checkQuantity(newQty, newQty); err != nil {
			return nil, err
		}
	}

	var prevBid, prevAsk Decimal
	if ob.emitter != nil {
		prevBid, prevAsk = ob.bboSnapshot()
	}

	inPlace := newPrice.Equal(order.price) && newQty.LessThanOrEqual(order.quantity)
	if inPlace {
		if order.Side() == Buy {
			ob.bids.DecreaseOrderQty(order, newQty)
		} else {
			ob.asks.DecreaseOrderQty(order, newQty)
		}
		if ob.emitter != nil {
			side_ := ob.sideFor(order.Side())
			ob.emit(EventVolume, VolumeEventData{Side: order.Side(), Price: order.price, Volume: side_.VolumeAt(order.price)})
		}
		ob.emitBBOIfChanged(prevBid, prevAsk)
		return nil, nil
	}

	// Re-queue path: remove and re-submit as a fresh limit order at the back of the new level.
	ob.deleteOrder(order)
	order.price = newPrice
	order.SetQuantity(newQty)
	order.originalQty = newQty
	order.canceled = false

	done, _ := ob.processLimitOrder(order)
	ob.emitBBOIfChanged(prevBid, prevAsk)
	return done, nil
}

// ModifyStop changes the stop price, limit price, and/or quantity of a resting stop order.
// The limit price is ignored for stop-market orders.
// Returns false if the order is not found or is not a stop-limit order.
// The order is removed from the stop book and re-inserted at the back of the new stop price bucket.
func (ob *OrderBook) ModifyStop(orderID string, newStopPrice, newLimitPrice, newQty Decimal) error {
	if ob.wal != nil {
		return ob.modifyStopLogged(orderID, newStopPrice, newLimitPrice, newQty)
	}
	return ob.modifyStop(orderID, newStopPrice, newLimitPrice, newQty)
}

// modifyStop is ModifyStop without the WAL gate.
func (ob *OrderBook) modifyStop(orderID string, newStopPrice, newLimitPrice, newQty Decimal) error {
	if ob.halted {
		return ErrTradingHalted
	}
	order := ob.GetOrder(orderID)
	if order == nil {
		return ErrOrderNotFound
	}
	if !order.IsStopOrder() {
		return ErrNotStopOrder
	}
	if ob.instrument != nil {
		if err := ob.instrument.checkPrice(newStopPrice); err != nil {
			return err
		}
		if err := ob.instrument.checkQuantity(newQty, newQty); err != nil {
			return err
		}
		if order.orderType == TypeStopLimit {
			if err := ob.instrument.checkPrice(newLimitPrice); err != nil {
				return err
			}
		}
	}
	ob.Stop.Remove(order)
	order.stop = newStopPrice
	if order.orderType == TypeStopLimit {
		order.price = newLimitPrice
	}
	order.SetQuantity(newQty)
	order.originalQty = newQty
	ob.Stop.Append(order)
	return nil
}

// validate runs every order check that Process can reject on, without changing the book or the order.
// ErrOrderExists is not included.
func (ob *OrderBook) validate(order *Order) error {
	if ob.halted {
		return ErrTradingHalted
	}
	if order == nil || (!order.IsMarketOrder() && !order.IsLimitOrder() && !order.IsStopOrder()) {
		return ErrInvalidOrder
	}
	if order.Side() != Buy && order.Side() != Sell {
		return ErrInvalidSide
	}
	if order.IsCanceled() {
		return ErrCanceledOrder
	}
	if order.Quantity().LessThanOrEqual(Zero) {
		return ErrInvalidQuantity
	}
	if !checkPrecision(order.Quantity(), ob.quantityPrecision) {
		return errQuantityPrecision(order.Quantity(), ob.quantityPrecision)
	}
	if !order.IsMarketOrder() && !checkPrecision(order.Price(), ob.pricePrecision) {
		return errPricePrecision(order.Price(), ob.pricePrecision)
	}
	if !order.IsMarketOrder() && order.IsStopOrder() && !checkPrecision(order.StopPrice(), ob.pricePrecision) {
		return errPricePrecision(order.StopPrice(), ob.pricePrecision)
	}
	if ob.instrument != nil {
		if err := ob.instrument.checkOrder(order); err != nil {
			return err
		}
	}
	if order.tif == FOK && order.extra().stp != STPNone && order.extra().participant != "" {
		return ErrInvalidSTP
	}
	if order.extra().ttl > 0 && (order.IsMarketOrder() || (order.tif != "" && order.tif != GTC)) {
		return ErrInvalidTif
	}
	if order.tif == GTD && !order.extra().expiresAt.After(ob.clock()) {
		return ErrOrderExpired
	}
	return nil
}

// Replace atomically swaps the resting order oldID for newOrder: the new order is fully
// validated before the old one is touched, so on error the old order stays on the book
// untouched. newOrder may reuse oldID. The new order goes to the back of its queue and
// matches like any submitted order; oldID is listed first in the returned Done.Canceled.
func (ob *OrderBook) Replace(oldID string, newOrder *Order) (*Done, error) {
	if ob.wal != nil {
		return ob.replaceLogged(oldID, newOrder)
	}
	return ob.replace(oldID, newOrder)
}

// replace is Replace without the WAL gate.
func (ob *OrderBook) replace(oldID string, newOrder *Order) (*Done, error) {
	old := ob.GetOrder(oldID)
	if old == nil {
		return nil, ErrOrderNotFound
	}
	if err := ob.validate(newOrder); err != nil {
		return nil, err
	}
	if newOrder.ID() != oldID && ob.GetOrder(newOrder.ID()) != nil {
		return nil, ErrOrderExists
	}
	ob.cancelOrder(oldID)
	done, err := ob.process(newOrder)
	if done != nil {
		done.Canceled = append([]string{oldID}, done.Canceled...)
	}
	return done, err
}

// Process public method
func (ob *OrderBook) Process(order *Order) (done *Done, err error) {
	if ob.wal != nil {
		return ob.processLogged(order)
	}
	return ob.process(order)
}

// process is Process without the WAL gate.
func (ob *OrderBook) process(order *Order) (done *Done, err error) {
	if err := ob.validate(order); err != nil {
		return nil, err
	}
	if ob.GetOrder(order.ID()) != nil {
		return nil, ErrOrderExists
	}

	if order.ext != nil && order.ext.ttl > 0 {
		order.tif = GTD
		order.ext.expiresAt = ob.clock().Add(order.ext.ttl)
		order.ext.ttl = 0
	}

	var expired []string
	if !ob.nextExpiry.IsZero() || order.tif == GTD {
		now := ob.clock()
		if !ob.nextExpiry.IsZero() && !now.Before(ob.nextExpiry) {
			for _, o := range ob.expireOrders(now) {
				expired = append(expired, o.ID())
			}
		}
		if order.tif == GTD && (ob.nextExpiry.IsZero() || order.extra().expiresAt.Before(ob.nextExpiry)) {
			ob.nextExpiry = order.extra().expiresAt
		}
	}

	if order.createdAt.IsZero() {
		order.createdAt = ob.clock()
	}
	ob.sequence++
	var prevBid, prevAsk Decimal
	if ob.emitter != nil {
		prevBid, prevAsk = ob.bboSnapshot()
	}

	if order.IsMarketOrder() {
		done, err = ob.processMarketOrder(order)
	} else if order.IsLimitOrder() {
		done, err = ob.processLimitOrder(order)
	} else {
		done, err = ob.processStopOrder(order)
	}

	if done != nil {
		done.Sequence = ob.sequence
		if len(expired) > 0 {
			done.Canceled = append(expired, done.Canceled...)
		}
	}
	ob.emitBBOIfChanged(prevBid, prevAsk)
	return done, err
}

// CalculateMarketPrice returns total market Price for requested quantity
func (ob *OrderBook) CalculateMarketPrice(side Side, quantity Decimal) (price Decimal, err error) {
	price = Zero

	var orders *OrderSide
	if side == Buy {
		orders = ob.asks
	} else {
		orders = ob.bids
	}

	level := orders.BestPriceQueue()
	iter := orders.NextLevel

	for quantity.GreaterThan(Zero) && level != nil {
		levelVolume := level.Volume()
		levelPrice := level.Price()
		if quantity.GreaterThanOrEqual(levelVolume) {
			price = price.Add(levelPrice.Mul(levelVolume))
			quantity = quantity.Sub(levelVolume)
			level = iter(levelPrice)
		} else {
			price = price.Add(levelPrice.Mul(quantity))
			quantity = Zero
		}
	}

	if quantity.GreaterThan(Zero) {
		err = ErrInsufficientQuantity
	}

	return
}

// private methods

func (ob *OrderBook) deleteStopOrderByID(orderID string) *Order {
	return ob.Stop.RemoveByID(orderID)
}

func (ob *OrderBook) deleteOrder(order *Order) *Order {
	delete(ob.orders, order.ID())

	if order.Side() == Buy {
		ob.bids.Remove(order)
	}

	if order.Side() == Sell {
		ob.asks.Remove(order)
	}

	if ob.emitter != nil {
		side_ := ob.sideFor(order.Side())
		ob.emit(EventVolume, VolumeEventData{Side: order.Side(), Price: order.price, Volume: side_.VolumeAt(order.price)})
	}

	return order
}

func (ob *OrderBook) processMarketOrder(marketOrder *Order) (done *Done, err error) {
	quantity := marketOrder.quantity

	var (
		side *OrderSide
		iter func() *OrderQueue
	)

	if marketOrder.Side() == Buy {
		side = ob.asks
	} else {
		side = ob.bids
	}

	iter = side.BestPriceQueue

	done = newDone(marketOrder, side.Len() > 0)

	if ob.checkOCO(marketOrder, done) {
		return
	}

	for quantity.GreaterThan(Zero) && side.Len() > 0 && !done.stpHalt {
		bestPrice := iter()
		if marketOrder.IsQuote() {
			remaining := ob.processQueueQuote(bestPrice, quantity, done)
			if remaining.Equal(quantity) && bestPrice.Len() > 0 {
				break // The budget cannot buy one more base unit at the best price.
			}
			quantity = remaining
		} else {
			quantity = ob.processQueue(bestPrice, quantity, done)
		}
	}

	done.setLeftQuantity(&quantity)

	if done.Processed.GreaterThan(Zero) {
		ob.appendToOCO(marketOrder, done)
	}

	// Market-to-limit: rest the remainder at the last trade price
	if marketOrder.extra().toLimit && !done.stpHalt && done.Left.GreaterThan(Zero) && done.Processed.GreaterThan(Zero) {
		if marketOrder.ext != nil {
			marketOrder.ext.toLimit = false
		}
		marketOrder.orderType = TypeLimit
		marketOrder.price = done.Trades[len(done.Trades)-1].Price
		marketOrder.SetQuantity(done.Left)
		ob.appendLimitOrder(marketOrder)
		done.Stored = true
	} else if done.Left.GreaterThan(Zero) || done.stpHalt {
		// If market GetOrder was not fulfilled then cancel it
		marketOrder.Cancel()
		done.appendCanceled(marketOrder)
		ob.emit(EventCanceled, marketOrder.ID())
	}

	ob.processPendingActivations(done)

	return done, nil
}

func (ob *OrderBook) processLimitOrder(limitOrder *Order) (done *Done, err error) {

	quantity := limitOrder.Quantity()

	var (
		side       *OrderSide
		comparator func(Decimal) bool
		iter       func() *OrderQueue
	)

	if limitOrder.Side() == Buy {
		side = ob.asks
		comparator = limitOrder.price.GreaterThanOrEqual
	} else {
		side = ob.bids
		comparator = limitOrder.price.LessThanOrEqual
	}

	iter = side.BestPriceQueue

	done = newDone(limitOrder, side.Len() > 0 && comparator(iter().Price()))

	if ob.checkOCO(limitOrder, done) {
		return
	}

	if limitOrder.IsPostOnly() {
		bestPrice := iter()
		if side.Len() > 0 && comparator(bestPrice.Price()) {
			limitOrder.Cancel()
			done.appendCanceled(limitOrder)
			ob.emit(EventCanceled, limitOrder.ID())
			return
		}
	}

	if limitOrder.TIF() == FOK {
		fokQty := quantity
		if limitOrder.IsIceberg() {
			fokQty = limitOrder.peakQty.Add(limitOrder.hiddenQty)
		}
		if !side.CanOrderBeFilled(limitOrder.Side(), limitOrder.price, fokQty) {
			limitOrder.Cancel()
			done.appendCanceled(limitOrder)
			ob.emit(EventCanceled, limitOrder.ID())
			return
		}
	}

	bestPrice := iter()

	for quantity.GreaterThan(Zero) && side.Len() > 0 && !done.stpHalt && comparator(bestPrice.Price()) {
		quantity = ob.processQueue(bestPrice, quantity, done)
		bestPrice = iter()
	}

	done.setLeftQuantity(&quantity)

	if done.stpHalt {
		limitOrder.Cancel()
		done.appendCanceled(limitOrder)
		ob.emit(EventCanceled, limitOrder.ID())
	} else if done.Left.GreaterThan(Zero) || done.Processed.Equal(Zero) {
		if done.Left.GreaterThan(Zero) {
			limitOrder.SetQuantity(done.Left)
		} else {
			limitOrder.SetQuantity(done.Quantity)
		}
		ob.appendLimitOrder(limitOrder)
		done.Stored = true
	} else {
		ob.appendToOCO(limitOrder, done)
	}

	// If IOC GetOrder was not fulfilled then cancel it
	if limitOrder.TIF() == IOC && quantity.GreaterThan(Zero) && !done.stpHalt {
		limitOrder.SetTaker()
		done.appendCanceled(ob.cancelOrder(limitOrder.ID()))
		done.Stored = false
	}

	ob.processPendingActivations(done)

	return
}

func (ob *OrderBook) processStopOrder(stopOrder *Order) (done *Done, err error) {
	ob.Stop.Append(stopOrder)
	ob.orders[stopOrder.ID()] = stopOrder
	done = newDone(stopOrder, false)
	return
}

func (ob *OrderBook) appendLimitOrder(order *Order) {
	if order.Side() == Buy {
		ob.bids.Append(order)
	}
	if order.Side() == Sell {
		ob.asks.Append(order)
	}

	ob.orders[order.ID()] = order

	// volume event: order now rests at price
	if ob.emitter != nil {
		side_ := ob.sideFor(order.Side())
		ob.emit(EventVolume, VolumeEventData{Side: order.Side(), Price: order.price, Volume: side_.VolumeAt(order.price)})
	}
}

func (ob *OrderBook) activateStopOrders(price Decimal) []*Order {
	var activated []*Order
	for _, slice := range [][]Order{ob.Stop.Activate(price, Buy), ob.Stop.Activate(price, Sell)} {
		for i := range slice {
			order := &slice[i]
			order.ActivateStopOrder()
			activated = append(activated, order)
		}
	}
	return activated
}

func (ob *OrderBook) processPendingActivations(done *Done) {
	for _, order := range done.drainPendingActivated() {
		var activatedDone *Done
		if order.IsMarketOrder() {
			// the stop book entry is a stale copy; market orders never rest, so drop it
			delete(ob.orders, order.ID())
			activatedDone, _ = ob.processMarketOrder(order)
		} else {
			activatedDone, _ = ob.processLimitOrder(order)
		}
		done.Activated = append(done.Activated, order.ID())
		done.Activations = append(done.Activations, activatedDone)
		ob.emit(EventActivated, order.ID())
	}
}

func (ob *OrderBook) processQueueQuote(bestPrice *OrderQueue, quantity Decimal, done *Done) Decimal {
	baseQuantity := ob.adaptQuantityBase(quantity, bestPrice.Price())
	remaining := ob.processQueue(bestPrice, baseQuantity, done)
	spent := ob.adaptQuantityQuote(baseQuantity.Sub(remaining), bestPrice.Price())
	return quantity.Sub(spent)
}

func (ob *OrderBook) processQueue(orderQueue *OrderQueue, quantity Decimal, done *Done) Decimal {
	touch := false
	price := orderQueue.Price()
	taker := done.Order.extra()
	checkSTP, takerParticipant := taker.stp != STPNone && taker.participant != "", taker.participant

	for quantity.GreaterThan(Zero) && orderQueue.Len() > 0 {
		o := orderQueue.First()
		if checkSTP && o.extra().participant == takerParticipant {
			if ob.preventSelfTrade(o, orderQueue, &quantity, done) {
				break
			}
			continue
		}
		touch = true
		orderQuantity := o.Quantity()
		if quantity.LessThan(orderQuantity) {
			done.appendOrder(o, quantity, price)
			orderQueue.UpdateVolume(quantity)
			o.DecreaseQuantity(quantity)
			ob.tradeID++
			ob.emitTrade(o, done.Order, price, quantity, orderQueue)
			quantity = Zero
		} else {
			ob.appendToOCO(o, done)
			ob.deleteOrder(o)
			done.appendOrder(o, orderQuantity, price)
			ob.tradeID++
			ob.emitTrade(o, done.Order, price, orderQuantity, orderQueue)
			quantity = quantity.Sub(orderQuantity)
			if o.IsIceberg() && o.hiddenQty.GreaterThan(Zero) {
				refillQty := o.peakQty
				if o.hiddenQty.LessThan(refillQty) {
					refillQty = o.hiddenQty
				}
				o.hiddenQty = o.hiddenQty.Sub(refillQty)
				o.quantity = refillQty
				o.originalQty = refillQty
				ob.appendLimitOrder(o)
			}
		}
	}

	if touch {
		if ob.breakerPct.GreaterThan(Zero) {
			ob.checkBreaker(price)
		}
		ob.trailStops(price)
		done.addPendingActivated(ob.activateStopOrders(price)...)
	}

	return quantity
}

// emitTrade publishes the trade and the maker level's new volume; it builds nothing when no emitter is attached.
func (ob *OrderBook) emitTrade(maker, taker *Order, price, amount Decimal, level *OrderQueue) {
	if ob.emitter == nil {
		return
	}
	ob.emit(EventTrade, TradeEventData{TradeID: ob.tradeID, MakerID: maker.ID(), TakerID: taker.ID(), Price: price, Amount: amount, Sequence: ob.sequence})
	ob.emit(EventVolume, VolumeEventData{Side: maker.Side(), Price: price, Volume: level.Volume()})
}

// trailStops moves trailing stops toward a trade at price; they never move away.
func (ob *OrderBook) trailStops(price Decimal) {
	var trailing []*Order
	for _, o := range ob.Stop.orders {
		if o.IsTrailing() {
			trailing = append(trailing, o)
		}
	}
	if len(trailing) > 1 { // map order is random; keep re-queueing deterministic for WAL replay
		sort.Slice(trailing, func(i, j int) bool {
			if !trailing[i].createdAt.Equal(trailing[j].createdAt) {
				return trailing[i].createdAt.Before(trailing[j].createdAt)
			}
			return trailing[i].id < trailing[j].id
		})
	}
	for _, o := range trailing {
		offset := o.extra().trail
		if o.extra().trailPct {
			offset = price.Mul(o.extra().trail).Div(FromInt(100))
		}
		var candidate Decimal
		if o.Side() == Sell {
			candidate = price.Sub(offset)
			if !candidate.GreaterThan(o.stop) {
				continue
			}
		} else {
			candidate = price.Add(offset)
			if !candidate.LessThan(o.stop) {
				continue
			}
		}
		ob.Stop.Remove(o)
		o.stop = candidate
		ob.Stop.Append(o)
	}
}

// preventSelfTrade applies the incoming order's STP mode against resting, a
// same-participant order. It returns true when the incoming order must stop matching.
func (ob *OrderBook) preventSelfTrade(resting *Order, queue *OrderQueue, quantity *Decimal, done *Done) bool {
	mode := done.Order.extra().stp
	if mode == STPDecrement && done.Order.IsQuote() {
		mode = STPCancelNewest // a quote budget cannot be decremented in base units
	}
	switch mode {
	case STPCancelOldest, STPCancelBoth:
		ob.cancelResting(resting, done)
		if mode == STPCancelBoth {
			done.stpHalt = true
		}
	case STPDecrement:
		overlap := *quantity
		if resting.Quantity().LessThan(overlap) {
			overlap = resting.Quantity()
		}
		*quantity = quantity.Sub(overlap)
		done.Quantity = done.Quantity.Sub(overlap)
		if overlap.Equal(resting.Quantity()) {
			ob.cancelResting(resting, done)
		} else {
			queue.UpdateVolume(overlap)
			resting.DecreaseQuantity(overlap)
			ob.emit(EventVolume, VolumeEventData{Side: resting.Side(), Price: queue.Price(), Volume: queue.Volume()})
		}
		done.stpHalt = quantity.Equal(Zero)
	default:
		done.stpHalt = true
	}
	return done.stpHalt
}

func (ob *OrderBook) cancelResting(o *Order, done *Done) {
	o.Cancel()
	ob.deleteOrder(o)
	done.appendCanceled(o)
	ob.emit(EventCanceled, o.ID())
}

// bboSnapshot captures the current best bid and ask prices (zero if empty).
func (ob *OrderBook) bboSnapshot() (bidPrice, askPrice Decimal) {
	if b := ob.bids.BestPriceQueue(); b != nil {
		bidPrice = b.Price()
	}
	if a := ob.asks.BestPriceQueue(); a != nil {
		askPrice = a.Price()
	}
	return
}

// emitBBOIfChanged emits EventBBO when the best bid or ask changed since prevBid/prevAsk.
func (ob *OrderBook) emitBBOIfChanged(prevBid, prevAsk Decimal) {
	if ob.emitter == nil {
		return
	}
	newBid, newAsk := ob.bboSnapshot()
	if newBid.Equal(prevBid) && newAsk.Equal(prevAsk) {
		return
	}
	ob.emit(EventBBO, BBOEventData{Bid: ob.BestBid(), Ask: ob.BestAsk()})
}

func (ob *OrderBook) sideFor(side Side) *OrderSide {
	if side == Buy {
		return ob.bids
	}
	return ob.asks
}

func (ob *OrderBook) adaptQuantityBase(quantity, price Decimal) Decimal {
	return quantity.Div(price)
}

func (ob *OrderBook) adaptQuantityQuote(quantity, price Decimal) Decimal {
	return quantity.Mul(price)
}

// checkOCO removes Order if OCO reference is already Processed
func (ob *OrderBook) checkOCO(order *Order, done *Done) bool {
	if order.oco == "" {
		return false
	}
	_, ok := ob.OCO[order.ID()]
	if !ok {
		return false
	}

	delete(ob.OCO, order.ID())

	ob.cancelOCO(order.ID(), done)

	return true
}

// appendToOCO appends Processed OCO Order
func (ob *OrderBook) appendToOCO(order *Order, done *Done) {
	if order.oco == "" {
		return
	}
	ob.OCO[order.OCO()] = struct{}{}

	ob.cancelOCO(order.OCO(), done)
}

func (ob *OrderBook) cancelOCO(orderID string, done *Done) {
	canceledOrder := ob.GetOrder(orderID)
	if canceledOrder == nil {
		return
	}

	if canceledOrder.IsStopOrder() {
		ob.deleteStopOrderByID(orderID)
		delete(ob.orders, orderID)
	} else {
		ob.deleteOrder(canceledOrder)
	}

	canceledOrder.Cancel()
	delete(ob.OCO, orderID)
	done.appendCanceled(canceledOrder)
	ob.emit(EventCanceled, orderID)
}

// BestBid returns the highest BUY price level and its total volume, or nil if empty.
func (ob *OrderBook) BestBid() *Level {
	q := ob.bids.BestPriceQueue()
	if q == nil {
		return nil
	}
	return &Level{Price: q.Price(), Volume: q.Volume()}
}

// BestAsk returns the lowest SELL price level and its total volume, or nil if empty.
func (ob *OrderBook) BestAsk() *Level {
	q := ob.asks.BestPriceQueue()
	if q == nil {
		return nil
	}
	return &Level{Price: q.Price(), Volume: q.Volume()}
}

// Level returns the total open amount at the given price on the given side, or nil if none.
func (ob *OrderBook) Level(side Side, price Decimal) (Decimal, bool) {
	var side_ *OrderSide
	if side == Buy {
		side_ = ob.bids
	} else {
		side_ = ob.asks
	}
	prices := side_.Prices()
	for _, p := range prices {
		if p.Equal(price) {
			return side_.prices[p].Volume(), true
		}
	}
	return Zero, false
}

// String implements fmt.Stringer interface
func (ob *OrderBook) String() string {
	builder := strings.Builder{}

	builder.WriteString("Ask:")
	builder.WriteString(ob.asks.String())
	builder.WriteString("\n")

	builder.WriteString("Bid:")
	builder.WriteString(ob.bids.String())
	builder.WriteString("\n")

	return builder.String()
}
