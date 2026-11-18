package matchingo

import (
	"fmt"
	"sort"
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

// Activate activates stop orders triggered by a trade at price on the given side.
// BUY stops activate when stop price ≤ trade price (lowest first).
// SELL stops activate when stop price ≥ trade price (highest first).
// Within the same stop price, insertion order is preserved.
func (sb *StopBook) Activate(price Decimal, side Side) []Order {
	var keys []Decimal
	for p := range sb.prices {
		if side == Buy && p.LessThanOrEqual(price) {
			keys = append(keys, p)
		} else if side == Sell && p.GreaterThanOrEqual(price) {
			keys = append(keys, p)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Slice(keys, func(i, j int) bool {
		if side == Buy {
			return keys[i].LessThan(keys[j])
		}
		return keys[i].GreaterThan(keys[j])
	})

	var result []Order
	for _, p := range keys {
		queue := sb.prices[p]
		n := queue.Len()
		kept := make([]*Order, 0, n)
		for i := 0; i < n; i++ {
			o := queue.Orders.PopFront()
			if o.Side() == side {
				result = append(result, *o)
				delete(sb.orders, o.ID())
				sb.numOrders--
			} else {
				kept = append(kept, o)
			}
		}
		queue.volume = Zero
		for _, o := range kept {
			queue.Orders.PushBack(o)
			queue.volume = queue.volume.Add(o.Quantity())
		}
		if queue.Len() == 0 {
			delete(sb.prices, p)
		}
	}
	return result
}

// inOrder returns all stop orders by ascending stop price, then queue order.
func (sb *StopBook) inOrder() []*Order {
	prices := make([]Decimal, 0, len(sb.prices))
	for p := range sb.prices {
		prices = append(prices, p)
	}
	sort.Slice(prices, func(i, j int) bool { return prices[i].LessThan(prices[j]) })
	out := make([]*Order, 0, sb.numOrders)
	for _, p := range prices {
		q := sb.prices[p]
		for i := 0; i < q.Orders.Len(); i++ {
			out = append(out, q.Orders.At(i))
		}
	}
	return out
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
