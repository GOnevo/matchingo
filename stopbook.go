package matchingo

import (
	"fmt"
	"strings"
)

// StopBook implements facade to operations with Stop Orders
type StopBook struct {
	prices    map[Decimal]*OrderQueue
	orders    map[string]*Order
	numOrders int
}

// NewStopBook creates new OrderSide manager
func NewStopBook() *StopBook {
	return &StopBook{
		prices: map[Decimal]*OrderQueue{},
		orders: map[string]*Order{},
	}
}

// Len returns amount of Orders
func (sb *StopBook) Len() int {
	return sb.numOrders
}

// Append appends Order to definite Price level
func (sb *StopBook) Append(o *Order) {
	_, ok := sb.orders[o.ID()]
	if ok {
		return
	}

	price := o.StopPrice()

	priceQueue, ok := sb.prices[price]
	if !ok {
		priceQueue = NewOrderQueue(price)
		sb.prices[price] = priceQueue
	}
	priceQueue.Append(o)
	sb.orders[o.ID()] = o
	sb.numOrders++
}

// Activate Orders by Stop Price
func (sb *StopBook) Activate(price Decimal) []Order {

	priceQueue, ok := sb.prices[price]
	if !ok {
		return nil
	}

	slice := priceQueue.Slice()
	for _, order := range slice {
		delete(sb.orders, order.ID())
	}

	if priceQueue.Len() == 0 {
		delete(sb.prices, price)
	}

	sb.numOrders = sb.numOrders - len(slice)
	return slice
}

// Remove removes Order from definite Price level
func (sb *StopBook) Remove(order *Order) *Order {
	price := order.StopPrice()

	priceQueue := sb.prices[price]
	priceQueue.Remove(order)

	if priceQueue.Len() == 0 {
		delete(sb.prices, price)
	}

	delete(sb.orders, order.ID())
	sb.numOrders--
	return order
}

// RemoveByID removes Order by ID
func (sb *StopBook) RemoveByID(id string) *Order {

	order, ok := sb.orders[id]
	if !ok {
		return nil
	}

	price := order.StopPrice()

	priceQueue, ok := sb.prices[price]
	if ok {
		priceQueue.Remove(order)
		if priceQueue.Len() == 0 {
			delete(sb.prices, price)
		}
	}

	delete(sb.orders, id)
	sb.numOrders--
	return order
}

// String implements fmt.Stringer interface
func (sb *StopBook) String() string {
	builder := strings.Builder{}

	for price, queue := range sb.prices {
		builder.WriteString(
			fmt.Sprintf(
				"\n%s -> size: %d",
				price,
				queue.Len(),
			),
		)
	}

	return builder.String()
}
