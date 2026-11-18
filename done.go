package matchingo

import (
	"encoding/json"
)

// Done structure
type Done struct {
	Order            *Order
	Trades           []TradeOrder
	Canceled         []string
	Activated        []string
	Activations      []*Done
	Stored           bool
	Sequence         uint64
	Quantity         Decimal
	Left             Decimal
	Processed        Decimal
	pendingActivated []*Order
	stpHalt          bool
}

// DoneJSON structure
type DoneJSON struct {
	Order       *TradeOrder  `json:"order"`
	Trades      []TradeOrder `json:"trades"`
	Canceled    []string     `json:"canceled"`
	Activated   []string     `json:"activated"`
	Activations []*Done      `json:"activations"`
	Left        string       `json:"left"`
	Processed   string       `json:"processed"`
	Stored      bool         `json:"stored"`
}

// newDone creates the result for order. When the order is known to cross the book, the Done is allocated
// together with room for a single fill, so recording it needs no second allocation.
func newDone(order *Order, crosses bool) *Done {
	var d *Done
	if crosses {
		w := &struct {
			Done
			trades [2]TradeOrder
		}{}
		w.Trades = w.trades[:0]
		d = &w.Done
	} else {
		d = &Done{}
	}
	d.Order = order
	d.Quantity = order.Quantity()
	d.Left = Zero
	d.Processed = Zero
	return d
}

// GetTradeOrder returns TradeOrder by id
func (d *Done) GetTradeOrder(id string) *TradeOrder {
	for i := range d.Trades {
		if d.Trades[i].OrderID == id {
			return &d.Trades[i]
		}
	}
	return nil
}

func (d *Done) appendOrder(order *Order, quantity, price Decimal) {

	if len(d.Trades) == 0 {
		if cap(d.Trades) < 2 { // crossing orders arrive with room for one fill
			d.Trades = make([]TradeOrder, 0, 2)
		}
		d.Trades = append(d.Trades, newTradeOrder(d.Order, Zero, d.Order.Price()))
	}

	d.Trades = append(d.Trades, newTradeOrder(order, quantity, price))
}

func (d *Done) tradesToSlice() []TradeOrder {
	return d.Trades
}

func (d *Done) appendCanceled(order *Order) {
	d.Canceled = append(d.Canceled, order.ID())
}

func (d *Done) appendActivated(order *Order) {
	d.Activated = append(d.Activated, order.ID())
}

func (d *Done) addPendingActivated(orders ...*Order) {
	d.pendingActivated = append(d.pendingActivated, orders...)
}

func (d *Done) drainPendingActivated() []*Order {
	pending := d.pendingActivated
	d.pendingActivated = nil
	return pending
}

func (d *Done) setLeftQuantity(quantity *Decimal) {
	if len(d.Trades) == 0 && !d.Order.IsMarketOrder() {
		return
	}
	d.Left = *quantity
	d.Processed = d.Quantity.Sub(d.Left)
	if len(d.Trades) != 0 {
		d.Trades[0].Quantity = d.Processed
	}
}

// MarshalJSON implements Marshaler interface
func (d *Done) MarshalJSON() ([]byte, error) {
	customStruct := struct {
		Order       *TradeOrder  `json:"order"`
		Trades      []TradeOrder `json:"trades"`
		Canceled    []string     `json:"canceled"`
		Activated   []string     `json:"activated"`
		Activations []*Done      `json:"activations"`
		Left        string       `json:"left"`
		Processed   string       `json:"processed"`
		Stored      bool         `json:"stored"`
		Sequence    uint64       `json:"sequence"`
	}{
		Order:       d.Order.ToSimple(),
		Trades:      d.tradesToSlice(),
		Canceled:    d.Canceled,
		Activated:   d.Activated,
		Activations: d.Activations,
		Left:        d.Left.String(),
		Processed:   d.Processed.String(),
		Stored:      d.Stored,
		Sequence:    d.Sequence,
	}
	return json.Marshal(customStruct)
}

// String implements Stringer interface
func (d *Done) String() string {
	j, _ := d.MarshalJSON()
	return string(j)
}
