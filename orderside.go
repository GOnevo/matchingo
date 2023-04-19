package matchingo

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-set"
	"github.com/shopspring/decimal"
)

// Level contains Price and Volume in depth
type Level struct {
	Price  decimal.Decimal `json:"Price"`
	Volume decimal.Decimal `json:"Volume"`
}

// OrderSide implements facade to operations with Order queue
type OrderSide struct {
	orderedPrices *set.TreeSet[decimal.Decimal, set.Compare[decimal.Decimal]]
	prices        map[string]*OrderQueue
	volume        decimal.Decimal
	numOrders     int
	depth         int
}

// NewOrderSideAsk creates new OrderSide manager
func NewOrderSideAsk() *OrderSide {
	return &OrderSide{
		orderedPrices: set.NewTreeSet[decimal.Decimal, set.Compare[decimal.Decimal]](func(a decimal.Decimal, b decimal.Decimal) int {
			return a.Cmp(b)
		}),
		prices: map[string]*OrderQueue{},
		volume: decimal.Zero,
	}
}

// NewOrderSideBid creates new OrderSide manager
func NewOrderSideBid() *OrderSide {
	return &OrderSide{
		orderedPrices: set.NewTreeSet[decimal.Decimal, set.Compare[decimal.Decimal]](func(a decimal.Decimal, b decimal.Decimal) int {
			return b.Cmp(a)
		}),
		prices: map[string]*OrderQueue{},
		volume: decimal.Zero,
	}
}

// Len returns amount of Orders
func (os *OrderSide) Len() int {
	return os.numOrders
}

// Depth returns depth of market
func (os *OrderSide) Depth() int {
	return os.depth
}

// Volume returns total amount of Volume in side
func (os *OrderSide) Volume() decimal.Decimal {
	return os.volume
}

// Append appends Order to definite Price level
func (os *OrderSide) Append(o *Order) {

	o.SetMaker()

	price := o.Price()
	strPrice := price.String()

	priceQueue, ok := os.prices[strPrice]
	if !ok {
		priceQueue = NewOrderQueue(price)
		os.prices[strPrice] = priceQueue
		os.orderedPrices.Insert(price)
		os.depth++
	}
	priceQueue.Append(o)
	os.numOrders++
	os.volume = os.volume.Add(o.Quantity())
}

// Remove removes Order from definite Price level
func (os *OrderSide) Remove(order *Order) *Order {
	price := order.Price()
	strPrice := price.String()

	priceQueue := os.prices[strPrice]
	priceQueue.Remove(order)

	if priceQueue.Len() == 0 {
		delete(os.prices, strPrice)
		os.orderedPrices.Remove(price)
		os.depth--
	}

	os.numOrders--
	os.volume = os.volume.Sub(order.Quantity())
	return order
}

// Prices returns slice of prices
func (os *OrderSide) Prices() []decimal.Decimal {
	return os.orderedPrices.Slice()
}

// CanOrderBeFilled checks FOK
func (os *OrderSide) CanOrderBeFilled(side Side, priceLevel, quantity decimal.Decimal) bool {
	if side == Buy {
		return os.CanBuyOrderBeFilled(priceLevel, quantity)
	}

	if side == Sell {
		return os.CanSellOrderBeFilled(priceLevel, quantity)
	}

	panic("unrecognized Order side")
}

// CanBuyOrderBeFilled checks FOK Orders
func (os *OrderSide) CanBuyOrderBeFilled(priceLevel, quantity decimal.Decimal) bool {

	if quantity.GreaterThan(os.Volume()) {
		return false
	}

	volume := decimal.Zero
	for _, price := range os.Prices() {
		if price.LessThanOrEqual(priceLevel) && volume.LessThan(quantity) {
			volume = volume.Add(os.prices[price.String()].Volume())
		} else {
			break
		}
	}

	return volume.GreaterThanOrEqual(quantity)
}

// CanSellOrderBeFilled checks FOK Orders
func (os *OrderSide) CanSellOrderBeFilled(priceLevel, quantity decimal.Decimal) bool {

	if quantity.GreaterThan(os.Volume()) {
		return false
	}

	volume := decimal.Zero
	for _, price := range os.Prices() {
		if price.GreaterThanOrEqual(priceLevel) && volume.LessThan(quantity) {
			volume = volume.Add(os.prices[price.String()].Volume())
		} else {
			break
		}
	}

	return volume.GreaterThanOrEqual(quantity)
}

// BestPriceQueue returns best Orders queue
func (os *OrderSide) BestPriceQueue() *OrderQueue {
	if os.depth > 0 && !os.orderedPrices.Empty() {
		return os.prices[os.orderedPrices.Min().String()]
	}
	return nil
}

// NextLevel returns next Orders queue after level
func (os *OrderSide) NextLevel(level decimal.Decimal) *OrderQueue {
	if os.depth > 0 && !os.orderedPrices.Empty() {
		price, ok := os.orderedPrices.FirstAbove(level)
		if !ok {
			return nil
		}
		return os.prices[price.String()]
	}
	return nil
}

// String implements fmt.Stringer interface
func (os *OrderSide) String() string {
	sb := strings.Builder{}

	for _, price := range os.orderedPrices.Slice() {
		sb.WriteString(
			fmt.Sprintf(
				"\n%s -> size: %d, volume: %s",
				price,
				os.prices[price.String()].Len(),
				os.prices[price.String()].Volume(),
			),
		)
	}

	return sb.String()
}
