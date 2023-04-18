package matchingo

import "github.com/shopspring/decimal"

type Done struct {
	Trade     *Trade
	Canceled  []string
	Activated []string
	Stored    bool
	Quantity  decimal.Decimal
	Left      decimal.Decimal
	Processed decimal.Decimal
}

func newDone(order *Order) *Done {
	return &Done{
		Trade:     NewTrade(order),
		Canceled:  []string{},
		Activated: []string{},
		Quantity:  order.OriginalQty(),
		Left:      decimal.Zero,
		Processed: decimal.Zero,
	}
}

func (d *Done) AppendOrder(order *Order, quantity, price decimal.Decimal) {
	d.Trade.Append(order, quantity, price)
}

func (d *Done) AppendCanceled(order *Order) {
	d.Canceled = append(d.Canceled, order.ID())
}

func (d *Done) AppendActivated(order *Order) {
	d.Activated = append(d.Activated, order.ID())
}

func (d *Done) SetLeftQuantity(quantity *decimal.Decimal) {
	if len(d.Trade.Orders) == 0 {
		return
	}
	d.Left = *quantity
	d.Processed = d.Quantity.Sub(d.Left)
}
