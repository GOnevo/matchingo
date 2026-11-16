package matchingo

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-set/v3"
)

// OrderSide implements facade to operations with Order queue
type OrderSide struct {
	orderedPrices *set.TreeSet[Decimal]
	prices        map[Decimal]*OrderQueue
	len           int
	depth         int
}

// NewOrderSideAsk creates new OrderSide manager
func NewOrderSideAsk() *OrderSide {
	return &OrderSide{
		orderedPrices: set.NewTreeSet[Decimal](func(a Decimal, b Decimal) int {
			return a.Compare(b)
		}),
		prices: map[Decimal]*OrderQueue{},
	}
}

// NewOrderSideBid creates new OrderSide manager
func NewOrderSideBid() *OrderSide {
	return &OrderSide{
		orderedPrices: set.NewTreeSet[Decimal](func(a Decimal, b Decimal) int {
			return b.Compare(a)
		}),
		prices: map[Decimal]*OrderQueue{},
	}
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
		priceQueue = NewOrderQueue(price)
		os.prices[price] = priceQueue
		os.orderedPrices.Insert(price)
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
		os.orderedPrices.Remove(price)
		os.depth--
	}

	os.len--
	return order
}

// Prices returns slice of prices
func (os *OrderSide) Prices() []Decimal {
	return os.orderedPrices.Slice()
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

// BestPriceQueue returns best Orders queue
func (os *OrderSide) BestPriceQueue() *OrderQueue {
	if os.depth > 0 && !os.orderedPrices.Empty() {
		return os.prices[os.orderedPrices.Min()]
	}
	return nil
}

// NextLevel returns next Orders queue after level
func (os *OrderSide) NextLevel(level Decimal) *OrderQueue {
	if os.depth > 0 && !os.orderedPrices.Empty() {
		price, ok := os.orderedPrices.FirstAbove(level)
		if !ok {
			return nil
		}
		return os.prices[price]
	}
	return nil
}

// String implements fmt.Stringer interface
func (os *OrderSide) String() string {
	sb := strings.Builder{}

	for _, price := range os.orderedPrices.Slice() {
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
