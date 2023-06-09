package matchingo

import (
	"github.com/shopspring/decimal"
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

func (ob *OrderBook) processMarketOrder(marketOrder *Order) (done *Done, err error) {

	side := marketOrder.Side()
	quantity := marketOrder.quantity

	if quantity.Sign() <= 0 {
		return nil, ErrInvalidQuantity
	}

	var (
		iter          func() *OrderQueue
		sideToProcess *OrderSide
	)

	if side == Buy {
		iter = ob.asks.BestPriceQueue
		sideToProcess = ob.asks
	} else {
		iter = ob.bids.BestPriceQueue
		sideToProcess = ob.bids
	}

	done = newDone(marketOrder)

	for quantity.Sign() > 0 && sideToProcess.Len() > 0 {
		bestPrice := iter()
		if marketOrder.Side() == Buy {
			quantity = ob.processQueueQuote(bestPrice, quantity, done)
		} else {
			quantity = ob.processQueue(bestPrice, quantity, done)
		}
	}

	done.setLeftQuantity(&quantity)

	// If market Order was not fulfilled then cancel it
	if done.Left.Sign() == 1 {
		marketOrder.Cancel()
		done.appendCanceled(marketOrder)
	}

	return done, nil
}

func (ob *OrderBook) processLimitOrder(limitOrder *Order) (done *Done, err error) {

	quantity := limitOrder.Quantity()
	tif := limitOrder.TIF()

	if _, ok := ob.orders[limitOrder.ID()]; ok {
		return nil, ErrOrderExists
	}

	var (
		sideToProcess *OrderSide
		comparator    func(decimal.Decimal) bool
		iter          func() *OrderQueue
	)

	if limitOrder.Side() == Buy {
		sideToProcess = ob.asks
		comparator = limitOrder.price.GreaterThanOrEqual
		iter = ob.asks.BestPriceQueue
	} else {
		sideToProcess = ob.bids
		comparator = limitOrder.price.LessThanOrEqual
		iter = ob.bids.BestPriceQueue
	}

	done = newDone(limitOrder)

	if ob.checkOCO(limitOrder, done) {
		return
	}

	if tif == FOK {
		if !sideToProcess.CanOrderBeFilled(limitOrder.Side(), limitOrder.price, quantity) {
			limitOrder.Cancel()
			done.appendCanceled(limitOrder)
			return
		}
	}

	bestPrice := iter()

	for quantity.Sign() > 0 && sideToProcess.Len() > 0 && comparator(bestPrice.Price()) {
		quantity = ob.processQueue(bestPrice, quantity, done)
		bestPrice = iter()
	}

	done.setLeftQuantity(&quantity)

	if done.Left.Sign() > 0 || done.Processed.Equal(decimal.Zero) {
		if done.Left.Sign() > 0 {
			limitOrder.SetQuantity(done.Left)
		} else {
			limitOrder.SetQuantity(done.Quantity)
		}
		ob.appendLimitOrder(limitOrder)
		done.Stored = true
	} else {
		ob.appendToOCO(limitOrder, done)
	}

	// If IOC Order was not fulfilled then cancel it
	if tif == IOC && quantity.Sign() == 1 {
		canceledOrder := ob.CancelOrder(limitOrder.ID())
		done.appendCanceled(canceledOrder)
		done.Stored = false
	}

	return
}

func (ob *OrderBook) processStopOrder(stopOrder *Order) (done *Done, err error) {
	ob.Stop.Append(stopOrder)
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

func (ob *OrderBook) activateStopOrders(price decimal.Decimal) []*Order {
	var activated []*Order
	orders := ob.Stop.Activate(price)
	for _, order := range orders {
		order.ActivateStopOrder()
		ob.appendLimitOrder(&order)
		activated = append(activated, &order)
	}

	return activated
}

func (ob *OrderBook) processQueueQuote(bestPrice *OrderQueue, quantity decimal.Decimal, done *Done) decimal.Decimal {
	return ob.adaptQuantityQuote(
		ob.processQueue(bestPrice, ob.adaptQuantityBase(quantity, bestPrice.Price()), done), bestPrice.Price(),
	)
}

func (ob *OrderBook) processQueue(orderQueue *OrderQueue, quantity decimal.Decimal, done *Done) decimal.Decimal {
	touch := false
	price := orderQueue.Price()

	for quantity.Sign() > 0 && orderQueue.Len() > 0 {
		touch = true
		o := orderQueue.First()
		orderQuantity := o.Quantity()
		if quantity.LessThan(orderQuantity) {
			done.appendOrder(&o, quantity, price)
			orderQueue.DecreaseQuantity(o, quantity)
			quantity = decimal.Zero
		} else {
			ob.appendToOCO(&o, done)
			ob.DeleteOrder(&o)
			done.appendOrder(&o, orderQuantity, price)
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

func (ob *OrderBook) adaptQuantityBase(quantity, price decimal.Decimal) decimal.Decimal {
	return quantity.Div(price)
}

func (ob *OrderBook) adaptQuantityQuote(quantity, price decimal.Decimal) decimal.Decimal {
	return quantity.Mul(price)
}

// Order returns Order by id
func (ob *OrderBook) Order(orderID string) *Order {
	order, ok := ob.orders[orderID]
	if !ok {
		return nil
	}

	return order
}

// Depth returns Price levels and volume at Price level
func (ob *OrderBook) Depth() (asks, bids []Level) {
	level := ob.asks.BestPriceQueue()
	for level != nil {
		asks = append(asks, Level{
			Price:  level.Price(),
			Volume: level.Volume(),
		})
		level = ob.asks.NextLevel(level.Price())
	}

	level = ob.bids.BestPriceQueue()
	for level != nil {
		bids = append(bids, Level{
			Price:  level.Price(),
			Volume: level.Volume(),
		})
		level = ob.bids.NextLevel(level.Price())
	}
	return
}

// DeleteStopOrder removes Order from the Stop book
func (ob *OrderBook) DeleteStopOrder(order *Order) *Order {
	ob.Stop.Remove(order)
	return order
}

// DeleteStopOrderByID removes Order from the Stop book by ID
func (ob *OrderBook) DeleteStopOrderByID(orderID string) *Order {
	return ob.Stop.RemoveByID(orderID)
}

// DeleteOrder removes Order from the Order book
func (ob *OrderBook) DeleteOrder(order *Order) *Order {
	delete(ob.orders, order.ID())

	if order.Side() == Buy {
		ob.bids.Remove(order)
	}

	if order.Side() == Sell {
		ob.asks.Remove(order)
	}

	return order
}

// DeleteOrderByID removes Order with given ID from the Order book
func (ob *OrderBook) DeleteOrderByID(orderID string) *Order {
	order, ok := ob.orders[orderID]
	if !ok {
		return nil
	}

	return ob.DeleteOrder(order)
}

// CancelOrder removes Order with given ID from the Order book
func (ob *OrderBook) CancelOrder(orderID string) *Order {
	order, ok := ob.orders[orderID]
	if !ok {
		return nil
	}

	order.Cancel()

	return ob.DeleteOrder(order)
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
	canceledOrder := ob.DeleteStopOrderByID(orderID)
	if canceledOrder != nil {
		canceledOrder.Cancel()
		delete(ob.OCO, orderID)
		done.appendCanceled(canceledOrder)
	}

	canceledOrder = ob.DeleteOrderByID(orderID)
	if canceledOrder != nil {
		canceledOrder.Cancel()
		delete(ob.OCO, orderID)
		done.appendCanceled(canceledOrder)
	}
}

// CalculateMarketPrice returns total market Price for requested Volume
// if err is not nil Price returns total Price of all levels in side
func (ob *OrderBook) CalculateMarketPrice(side Side, quantity decimal.Decimal) (price decimal.Decimal, err error) {
	price = decimal.Zero

	var orders *OrderSide
	if side == Buy {
		orders = ob.asks
	} else {
		orders = ob.bids
	}

	level := orders.BestPriceQueue()
	iter := orders.NextLevel

	for quantity.Sign() > 0 && level != nil {
		levelVolume := level.Volume()
		levelPrice := level.Price()
		if quantity.GreaterThanOrEqual(levelVolume) {
			price = price.Add(levelPrice.Mul(levelVolume))
			quantity = quantity.Sub(levelVolume)
			level = iter(levelPrice)
		} else {
			price = price.Add(levelPrice.Mul(quantity))
			quantity = decimal.Zero
		}
	}

	if quantity.Sign() > 0 {
		err = ErrInsufficientQuantity
	}

	return
}

// String implements fmt.Stringer interface
func (ob *OrderBook) String() string {
	return "\nAsk:" + ob.asks.String() + "\n------------------------------------\nBid:" + ob.bids.String()
}
