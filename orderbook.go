package matchingo

import (
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
	quantityPrecision int // max decimal places for quantity; 0 = unlimited
	pricePrecision    int // max decimal places for price; 0 = unlimited
}

// Sequence returns the current monotonic sequence counter.
// It is incremented once per successful Process call.
func (ob *OrderBook) Sequence() uint64 { return ob.sequence }

// NewOrderBook creates Orderbook object
func NewOrderBook() *OrderBook {
	return &OrderBook{
		orders: map[string]*Order{},
		bids:   NewOrderSideBid(),
		asks:   NewOrderSideAsk(),
		Stop:   NewStopBook(),
		OCO:    map[string]struct{}{},
	}
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

// Cancel removes the order with the given ID and returns true on success, false if not found.
func (ob *OrderBook) Cancel(orderID string) bool {
	return ob.CancelOrder(orderID) != nil
}

// Modify changes the price or quantity of a resting limit order.
// Returns false if the order is not found or is a stop-limit order.
// In-place decrease: same price and newQty ≤ current open → keep time priority.
// Re-queue: price change or quantity increase → remove and re-insert at back of new price level, then match.
func (ob *OrderBook) Modify(orderID string, newPrice, newQty Decimal) (*Done, bool) {
	order := ob.GetOrder(orderID)
	if order == nil {
		return nil, false
	}
	if order.IsStopOrder() {
		return nil, false
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
		return nil, true
	}

	// Re-queue path: remove and re-submit as a fresh limit order at the back of the new level.
	ob.deleteOrder(order)
	order.price = newPrice
	order.SetQuantity(newQty)
	order.originalQty = newQty
	order.canceled = false

	done, _ := ob.processLimitOrder(order)
	ob.emitBBOIfChanged(prevBid, prevAsk)
	return done, true
}

// ModifyStop changes the stop price, limit price, and/or quantity of a resting stop-limit order.
// Returns false if the order is not found or is not a stop-limit order.
// The order is removed from the stop book and re-inserted at the back of the new stop price bucket.
func (ob *OrderBook) ModifyStop(orderID string, newStopPrice, newLimitPrice, newQty Decimal) bool {
	order := ob.GetOrder(orderID)
	if order == nil || !order.IsStopOrder() {
		return false
	}
	ob.Stop.Remove(order)
	order.stop = newStopPrice
	order.price = newLimitPrice
	order.SetQuantity(newQty)
	order.originalQty = newQty
	ob.Stop.Append(order)
	return true
}

// Process public method
func (ob *OrderBook) Process(order *Order) (done *Done, err error) {
	if order == nil || (!order.IsMarketOrder() && !order.IsLimitOrder() && !order.IsStopOrder()) {
		return nil, ErrInvalidOrder
	}
	if order.Side() != Buy && order.Side() != Sell {
		return nil, ErrInvalidSide
	}
	if order.IsCanceled() {
		return nil, ErrCanceledOrder
	}
	if order.Quantity().LessThanOrEqual(Zero) {
		return nil, ErrInvalidQuantity
	}
	if !checkPrecision(order.Quantity(), ob.quantityPrecision) {
		return nil, ErrQuantityPrecision
	}
	if !order.IsMarketOrder() && !checkPrecision(order.Price(), ob.pricePrecision) {
		return nil, ErrPricePrecision
	}
	if !order.IsMarketOrder() && order.IsStopOrder() && !checkPrecision(order.StopPrice(), ob.pricePrecision) {
		return nil, ErrPricePrecision
	}
	if ob.GetOrder(order.ID()) != nil {
		return nil, ErrOrderExists
	}

	if order.createdAt.IsZero() {
		order.createdAt = time.Now()
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

	done = newDone(marketOrder)

	for quantity.GreaterThan(Zero) && side.Len() > 0 {
		bestPrice := iter()
		if marketOrder.IsQuote() {
			remaining := ob.processQueueQuote(bestPrice, quantity, done)
			if remaining.Equal(quantity) {
				break // The budget cannot buy one more base unit at the best price.
			}
			quantity = remaining
		} else {
			quantity = ob.processQueue(bestPrice, quantity, done)
		}
	}

	done.setLeftQuantity(&quantity)

	// If market GetOrder was not fulfilled then cancel it
	if done.Left.GreaterThan(Zero) {
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

	done = newDone(limitOrder)

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

	for quantity.GreaterThan(Zero) && side.Len() > 0 && comparator(bestPrice.Price()) {
		quantity = ob.processQueue(bestPrice, quantity, done)
		bestPrice = iter()
	}

	done.setLeftQuantity(&quantity)

	if done.Left.GreaterThan(Zero) || done.Processed.Equal(Zero) {
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
	if limitOrder.TIF() == IOC && quantity.GreaterThan(Zero) {
		limitOrder.SetTaker()
		done.appendCanceled(ob.CancelOrder(limitOrder.ID()))
		done.Stored = false
	}

	ob.processPendingActivations(done)

	return
}

func (ob *OrderBook) processStopOrder(stopOrder *Order) (done *Done, err error) {
	ob.Stop.Append(stopOrder)
	ob.orders[stopOrder.ID()] = stopOrder
	done = newDone(stopOrder)
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
		activatedDone, _ := ob.processLimitOrder(order)
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

	for quantity.GreaterThan(Zero) && orderQueue.Len() > 0 {
		touch = true
		o := orderQueue.First()
		orderQuantity := o.Quantity()
		if quantity.LessThan(orderQuantity) {
			done.appendOrder(o, quantity, price)
			orderQueue.UpdateVolume(quantity)
			o.DecreaseQuantity(quantity)
			ob.tradeID++
			ob.emit(EventTrade, TradeEventData{TradeID: ob.tradeID, MakerID: o.ID(), TakerID: done.Order.ID(), Price: price, Amount: quantity, Sequence: ob.sequence})
			ob.emit(EventVolume, VolumeEventData{Side: o.Side(), Price: price, Volume: orderQueue.Volume()})
			quantity = Zero
		} else {
			ob.appendToOCO(o, done)
			ob.deleteOrder(o)
			done.appendOrder(o, orderQuantity, price)
			ob.tradeID++
			ob.emit(EventTrade, TradeEventData{TradeID: ob.tradeID, MakerID: o.ID(), TakerID: done.Order.ID(), Price: price, Amount: orderQuantity, Sequence: ob.sequence})
			ob.emit(EventVolume, VolumeEventData{Side: o.Side(), Price: price, Volume: orderQueue.Volume()})
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
		done.addPendingActivated(ob.activateStopOrders(price)...)
	}

	return quantity
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
func (ob *OrderBook) Level(side Side, price Decimal) *Decimal {
	var side_ *OrderSide
	if side == Buy {
		side_ = ob.bids
	} else {
		side_ = ob.asks
	}
	prices := side_.Prices()
	for _, p := range prices {
		if p.Equal(price) {
			q := side_.prices[p]
			v := q.Volume()
			return &v
		}
	}
	return nil
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
