package matchingo

import (
	"fmt"
	"sort"
	"strings"
)

// OrderSide implements facade to operations with Order queue
type OrderSide struct {
	orderedPrices []Decimal
	less          func(a, b Decimal) bool
	prices        map[Decimal]*OrderQueue
	len           int
	depth         int
	spare         *OrderQueue // emptied queue kept for the next new price level
}

// NewOrderSideAsk creates new OrderSide manager
func NewOrderSideAsk() *OrderSide {
	return &OrderSide{
		less:   func(a, b Decimal) bool { return a.Compare(b) < 0 },
		prices: map[Decimal]*OrderQueue{},
	}
}

// NewOrderSideBid creates new OrderSide manager
func NewOrderSideBid() *OrderSide {
	return &OrderSide{
		less:   func(a, b Decimal) bool { return b.Compare(a) < 0 },
		prices: map[Decimal]*OrderQueue{},
	}
}

func (os *OrderSide) priceInsert(p Decimal) {
	i := sort.Search(len(os.orderedPrices), func(i int) bool { return !os.less(os.orderedPrices[i], p) })
	if i < len(os.orderedPrices) && os.orderedPrices[i] == p {
		return
	}
	os.orderedPrices = append(os.orderedPrices, Decimal{})
	copy(os.orderedPrices[i+1:], os.orderedPrices[i:])
	os.orderedPrices[i] = p
}

func (os *OrderSide) priceRemove(p Decimal) {
	i := sort.Search(len(os.orderedPrices), func(i int) bool { return !os.less(os.orderedPrices[i], p) })
	if i >= len(os.orderedPrices) || os.orderedPrices[i] != p {
		return
	}
	os.orderedPrices = append(os.orderedPrices[:i], os.orderedPrices[i+1:]...)
}

func (os *OrderSide) priceMin() (Decimal, bool) {
	if len(os.orderedPrices) == 0 {
		return Zero, false
	}
	return os.orderedPrices[0], true
}

func (os *OrderSide) priceFirstAbove(p Decimal) (Decimal, bool) {
	i := sort.Search(len(os.orderedPrices), func(i int) bool { return os.less(p, os.orderedPrices[i]) })
	if i >= len(os.orderedPrices) {
		return Zero, false
	}
	return os.orderedPrices[i], true
}

// Len returns amount of Orders
func (os *OrderSide) Len() int {
	return os.len
}

// Depth returns depth of market
func (os *OrderSide) Depth() int {
	return os.depth
}

// Append appends Order to definite Price level
func (os *OrderSide) Append(o *Order) {

	o.SetMaker()

	price := o.Price()

	priceQueue, ok := os.prices[price]
	if !ok {
		if priceQueue = os.spare; priceQueue != nil {
			os.spare = nil
			priceQueue.price, priceQueue.volume = price, Zero
		} else {
			priceQueue = NewOrderQueue(price)
		}
		os.prices[price] = priceQueue
		os.priceInsert(price)
		os.depth++
	}
	priceQueue.Append(o)
	os.len++
}

// Remove removes Order from definite Price level, returning nil if it is absent.
func (os *OrderSide) Remove(order *Order) *Order {
	if order == nil {
		return nil
	}
	price := order.Price()

	priceQueue := os.prices[price]
	if priceQueue == nil || !priceQueue.Remove(order) {
		return nil
	}

	if priceQueue.Len() == 0 {
		delete(os.prices, price)
		os.priceRemove(price)
		os.spare = priceQueue
		os.depth--
	}

	os.len--
	return order
}

// DecreaseOrderQty decreases a resting order's quantity in-place, preserving time priority.
func (os *OrderSide) DecreaseOrderQty(order *Order, newQty Decimal) {
	priceQueue := os.prices[order.Price()]
	if priceQueue == nil {
		return
	}
	delta := order.quantity.Sub(newQty)
	priceQueue.UpdateVolume(delta)
	order.SetQuantity(newQty)
}

// Prices returns slice of prices
func (os *OrderSide) Prices() []Decimal {
	out := make([]Decimal, len(os.orderedPrices))
	copy(out, os.orderedPrices)
	return out
}

// CanOrderBeFilled checks FOK
func (os *OrderSide) CanOrderBeFilled(side Side, priceLevel, quantity Decimal) bool {
	if side == Buy {
		return os.CanBuyOrderBeFilled(priceLevel, quantity)
	}

	if side == Sell {
		return os.CanSellOrderBeFilled(priceLevel, quantity)
	}

	panic("unrecognized GetOrder side")
}

// CanBuyOrderBeFilled checks FOK Orders
func (os *OrderSide) CanBuyOrderBeFilled(priceLevel, quantity Decimal) bool {

	if os.Len() == 0 {
		return false
	}

	volume := Zero
	for _, price := range os.Prices() {
		if price.LessThanOrEqual(priceLevel) && volume.LessThan(quantity) {
			volume = volume.Add(os.prices[price].Volume())
		} else {
			break
		}
	}

	return volume.GreaterThanOrEqual(quantity)
}

// CanSellOrderBeFilled checks FOK Orders
func (os *OrderSide) CanSellOrderBeFilled(priceLevel, quantity Decimal) bool {

	if os.Len() == 0 {
		return false
	}

	volume := Zero
	for _, price := range os.Prices() {
		if price.GreaterThanOrEqual(priceLevel) && volume.LessThan(quantity) {
			volume = volume.Add(os.prices[price].Volume())
		} else {
			break
		}
	}

	return volume.GreaterThanOrEqual(quantity)
}

// VolumeAt returns the total volume at the given price, or Zero if none.
func (os *OrderSide) VolumeAt(price Decimal) Decimal {
	q, ok := os.prices[price]
	if !ok {
		return Zero
	}
	return q.Volume()
}

// BestPriceQueue returns best Orders queue
func (os *OrderSide) BestPriceQueue() *OrderQueue {
	if p, ok := os.priceMin(); ok {
		return os.prices[p]
	}
	return nil
}

// NextLevel returns next Orders queue after level
func (os *OrderSide) NextLevel(level Decimal) *OrderQueue {
	if price, ok := os.priceFirstAbove(level); ok {
		return os.prices[price]
	}
	return nil
}

// String implements fmt.Stringer interface
func (os *OrderSide) String() string {
	sb := strings.Builder{}

	for _, price := range os.orderedPrices {
		sb.WriteString(
			fmt.Sprintf(
				"\n%s -> orders: %d, volume: %s",
				price,
				os.prices[price].Len(),
				os.prices[price].Volume(),
			),
		)
	}

	return sb.String()
}
