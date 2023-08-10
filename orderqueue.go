package matchingo

import (
	"github.com/gammazero/deque"
	"github.com/nikolaydubina/fpdecimal"
)

// OrderQueue stores and manage chain of Orders
type OrderQueue struct {
	volume fpdecimal.Decimal
	price  fpdecimal.Decimal
	Orders *deque.Deque[*Order]
}

// NewOrderQueue creates and initialize OrderQueue object
func NewOrderQueue(price fpdecimal.Decimal) *OrderQueue {
	return &OrderQueue{
		price:  price,
		volume: fpdecimal.Zero,
		Orders: deque.New[*Order](),
	}
}

// Len returns amount of Orders in queue
func (oq *OrderQueue) Len() int {
	return oq.Orders.Len()
}

// Price returns Price level of the queue
func (oq *OrderQueue) Price() fpdecimal.Decimal {
	return oq.price
}

// Volume returns total Orders volume
func (oq *OrderQueue) Volume() fpdecimal.Decimal {
	return oq.volume
}

// First returns top Order in queue
func (oq *OrderQueue) First() *Order {
	return oq.Orders.Front()
}

// Append adds Order to tail of the queue
func (oq *OrderQueue) Append(o *Order) {
	oq.volume = oq.volume.Add(o.Quantity())
	oq.Orders.PushBack(o)
}

// UpdateVolume updates volume
func (oq *OrderQueue) UpdateVolume(o *Order) {
	oq.volume = oq.volume.Sub(o.Quantity())
}

// Remove removes Order from the queue
func (oq *OrderQueue) Remove(order *Order) bool {
	index := oq.Orders.Index(func(o *Order) bool {
		return o.ID() == order.ID()
	})
	return oq.RemoveIndex(index)
}

// RemoveByID removes Order from the queue
func (oq *OrderQueue) RemoveByID(id string) bool {
	index := oq.Orders.Index(func(order *Order) bool {
		return order.ID() == id
	})
	return oq.RemoveIndex(index)
}

// RemoveIndex removes Order from the queue
func (oq *OrderQueue) RemoveIndex(index int) bool {
	if index != -1 {
		order := oq.Orders.At(index)
		oq.volume = oq.volume.Sub(order.Quantity())
		oq.Orders.Remove(index)
		return true
	}

	return false
}

// Slice returns slice of Orders, queue will be empty
func (oq *OrderQueue) Slice() []Order {
	var slice []Order
	for oq.Orders.Len() > 0 {
		order := oq.Orders.PopFront()
		oq.volume = oq.volume.Sub(order.Quantity())
		slice = append(slice, *order)
	}
	return slice
}
