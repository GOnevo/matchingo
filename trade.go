package matchingo

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Participant structure
type Participant struct {
	OrderID     string
	Role        Role
	Price       decimal.Decimal
	Quantity    decimal.Decimal
	ReferenceID string
}

func (p *Participant) String() string {
	return "\t" + p.OrderID + "|price:" + p.Price.String() + "|q:" + p.Quantity.String() + "|role:" + string(p.Role) + "|" + p.ReferenceID
}

func newParticipant(order *Order, quantity, price decimal.Decimal, refID string) *Participant {
	return &Participant{
		OrderID:     order.ID(),
		Role:        order.Role(),
		Price:       price,
		Quantity:    quantity,
		ReferenceID: refID,
	}
}

// Trade structure
type Trade struct {
	Order  *Order
	Orders map[string]*Participant
}

// NewTrade public constructor
func NewTrade(order *Order) *Trade {
	return &Trade{
		Order:  order,
		Orders: map[string]*Participant{},
	}
}

// Append public method
func (t *Trade) Append(order *Order, quantity, price decimal.Decimal) {

	if _, ok := t.Orders[order.ID()]; ok {
		return
	}

	t.Orders[order.ID()] = newParticipant(order, quantity, price, t.Order.ID())
}

// String implements fmt.Stringer interface
func (t *Trade) String() string {
	sb := strings.Builder{}
	sb.WriteString("Orders:\n")
	for _, part := range t.Orders {
		sb.WriteString(part.String())
	}
	return t.Order.String() + sb.String() + "\n"
}
