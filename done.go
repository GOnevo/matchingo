package matchingo

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

// Done structure
type Done struct {
	Trade     *Trade
	Canceled  []string
	Activated []string
	Stored    bool
	Quantity  decimal.Decimal
	Left      decimal.Decimal
	Processed decimal.Decimal
}

type DoneJSON struct {
	Trade struct {
		Order  SimpleOrder   `json:"order"`
		Orders []Participant `json:"orders"`
	} `json:"trade"`
	Canceled  []string `json:"canceled"`
	Activated []string `json:"activated"`
	Stored    bool     `json:"stored"`
	Left      string   `json:"left"`
	Processed string   `json:"processed"`
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

func (d *Done) appendOrder(order *Order, quantity, price decimal.Decimal) {
	d.Trade.Append(order, quantity, price)
}

func (d *Done) appendCanceled(order *Order) {
	d.Canceled = append(d.Canceled, order.ID())
}

func (d *Done) appendActivated(order *Order) {
	d.Activated = append(d.Activated, order.ID())
}

func (d *Done) setLeftQuantity(quantity *decimal.Decimal) {
	if len(d.Trade.Orders) == 0 {
		return
	}
	d.Left = *quantity
	d.Processed = d.Quantity.Sub(d.Left)
}

// ToJSON returns Done structure as JSON string
func (d *Done) ToJSON() string {

	jsonStruct := DoneJSON{}
	jsonStruct.Trade.Order = *d.Trade.Order.ToSimple()
	jsonStruct.Trade.Orders = d.Trade.OrdersSlice()
	jsonStruct.Stored = d.Stored
	jsonStruct.Left = d.Left.String()
	jsonStruct.Processed = d.Processed.String()
	jsonStruct.Canceled = d.Canceled
	jsonStruct.Activated = d.Activated

	j, err := json.Marshal(jsonStruct)
	if err != nil {
		return ""
	}

	return string(j)
}
