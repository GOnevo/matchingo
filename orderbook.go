package matchingo

import (
	"strings"

	"github.com/nikolaydubina/fpdecimal"
)

// OrderBook implements standard matching algorithm
type OrderBook struct {
	orders map[string]*Order
	asks   *OrderSide
	bids   *OrderSide
	Stop   *StopBook
	OCO    map[string]struct{}
}

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

	order.Cancel()

	if order.IsStopOrder() {
		ob.Stop.Remove(order)
		delete(ob.orders, order.ID())
	} else {
		ob.deleteOrder(order)
	}

	return order
}

// Process public method
func (ob *OrderBook) Process(order *Order) (done *Done, err error) {
	if order.IsMarketOrder() {
		return ob.processMarketOrder(order)
	}

	if order.IsLimitOrder() {
		return ob.processLimitOrder(order)
	}

	if order.IsStopOrder() {
		return ob.processStopOrder(order)
	}

	panic("unrecognized order type")
}

// CalculateMarketPrice returns total market Price for requested quantity
func (ob *OrderBook) CalculateMarketPrice(side Side, quantity fpdecimal.Decimal) (price fpdecimal.Decimal, err error) {
	price = fpdecimal.Zero

	var orders *OrderSide
	if side == Buy {
		orders = ob.asks
	} else {
		orders = ob.bids
	}

	level := orders.BestPriceQueue()
	iter := orders.NextLevel

	for quantity.GreaterThan(fpdecimal.Zero) && level != nil {
		levelVolume := level.Volume()
		levelPrice := level.Price()
		if quantity.GreaterThanOrEqual(levelVolume) {
			price = price.Add(levelPrice.Mul(levelVolume))
			quantity = quantity.Sub(levelVolume)
			level = iter(levelPrice)
		} else {
			price = price.Add(levelPrice.Mul(quantity))
			quantity = fpdecimal.Zero
		}
	}

	if quantity.GreaterThan(fpdecimal.Zero) {
		err = ErrInsufficientQuantity
	}

	return
}

// private methods

func (ob *OrderBook) deleteStopOrder(order *Order) *Order {
	ob.Stop.Remove(order)
	return order
}

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

	return order
}

func (ob *OrderBook) deleteOrderByID(orderID string) *Order {
	order := ob.GetOrder(orderID)
	if order == nil {
		return nil
	}

	return ob.deleteOrder(order)
}

func (ob *OrderBook) processMarketOrder(marketOrder *Order) (done *Done, err error) {
	quantity := marketOrder.quantity

	if quantity.LessThanOrEqual(fpdecimal.Zero) {
		return nil, ErrInvalidQuantity
	}

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

	for quantity.GreaterThan(fpdecimal.Zero) && side.Len() > 0 {
		bestPrice := iter()
		if marketOrder.IsQuote() {
			quantity = ob.processQueueQuote(bestPrice, quantity, done)
		} else {
			quantity = ob.processQueue(bestPrice, quantity, done)
		}
	}

	done.setLeftQuantity(&quantity)

	// If market GetOrder was not fulfilled then cancel it
	if done.Left.GreaterThan(fpdecimal.Zero) {
		marketOrder.Cancel()
		done.appendCanceled(marketOrder)
	}

	return done, nil
}

func (ob *OrderBook) processLimitOrder(limitOrder *Order) (done *Done, err error) {

	quantity := limitOrder.Quantity()

	order := ob.GetOrder(limitOrder.ID())
	if order != nil {
		return nil, ErrOrderExists
	}

	var (
		side       *OrderSide
		comparator func(fpdecimal.Decimal) bool
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

	if limitOrder.TIF() == FOK {
		if !side.CanOrderBeFilled(limitOrder.Side(), limitOrder.price, quantity) {
			limitOrder.Cancel()
			done.appendCanceled(limitOrder)
			return
		}
	}

	bestPrice := iter()

	for quantity.GreaterThan(fpdecimal.Zero) && side.Len() > 0 && comparator(bestPrice.Price()) {
		quantity = ob.processQueue(bestPrice, quantity, done)
		bestPrice = iter()
	}

	done.setLeftQuantity(&quantity)

	if done.Left.GreaterThan(fpdecimal.Zero) || done.Processed.Equal(fpdecimal.Zero) {
		if done.Left.GreaterThan(fpdecimal.Zero) {
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
	if limitOrder.TIF() == IOC && quantity.GreaterThan(fpdecimal.Zero) {
		limitOrder.SetTaker()
		done.appendCanceled(ob.CancelOrder(limitOrder.ID()))
		done.Stored = false
	}

	return
}

func (ob *OrderBook) processStopOrder(stopOrder *Order) (done *Done, err error) {
	ob.Stop.Append(stopOrder)
	ob.orders[stopOrder.ID()] = stopOrder
	done = newDone(stopOrder)
	return
}

func (ob *OrderBook) appendLimitOrder(order *Order) {
	if order.IsLimitOrder() {
		if order.Side() == Buy {
			ob.bids.Append(order)
		}
		if order.Side() == Sell {
			ob.asks.Append(order)
		}

		ob.orders[order.ID()] = order

		return
	}

	panic("order has not LIMIT type")
}

func (ob *OrderBook) activateStopOrders(price fpdecimal.Decimal) []*Order {
	var activated []*Order
	orders := ob.Stop.Activate(price)
	for _, order := range orders {
		order.ActivateStopOrder()
		ob.appendLimitOrder(&order)
		activated = append(activated, &order)
	}

	return activated
}

func (ob *OrderBook) processQueueQuote(bestPrice *OrderQueue, quantity fpdecimal.Decimal, done *Done) fpdecimal.Decimal {
	return ob.adaptQuantityQuote(
		ob.processQueue(bestPrice, ob.adaptQuantityBase(quantity, bestPrice.Price()), done), bestPrice.Price(),
	)
}

func (ob *OrderBook) processQueue(orderQueue *OrderQueue, quantity fpdecimal.Decimal, done *Done) fpdecimal.Decimal {
	touch := false
	price := orderQueue.Price()

	for quantity.GreaterThan(fpdecimal.Zero) && orderQueue.Len() > 0 {
		touch = true
		o := orderQueue.First()
		orderQuantity := o.Quantity()
		if quantity.LessThan(orderQuantity) {
			done.appendOrder(o, quantity, price)
			o.DecreaseQuantity(quantity)
			orderQueue.UpdateVolume(o)
			quantity = fpdecimal.Zero
		} else {
			ob.appendToOCO(o, done)
			ob.deleteOrder(o)
			done.appendOrder(o, orderQuantity, price)
			quantity = quantity.Sub(orderQuantity)
		}
	}

	if touch {
		// activate Stop Orders for this Price level
		for _, activatedOrder := range ob.activateStopOrders(price) {
			done.appendActivated(activatedOrder)
		}
	}

	return quantity
}

func (ob *OrderBook) adaptQuantityBase(quantity, price fpdecimal.Decimal) fpdecimal.Decimal {
	return quantity.Div(price)
}

func (ob *OrderBook) adaptQuantityQuote(quantity, price fpdecimal.Decimal) fpdecimal.Decimal {
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
	canceledOrder := ob.deleteStopOrderByID(orderID)
	if canceledOrder != nil {
		canceledOrder.Cancel()
		delete(ob.OCO, orderID)
		done.appendCanceled(canceledOrder)
	}

	canceledOrder = ob.deleteOrderByID(orderID)
	if canceledOrder != nil {
		canceledOrder.Cancel()
		delete(ob.OCO, orderID)
		done.appendCanceled(canceledOrder)
	}
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
