package matchingo

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Order stores information about order
type Order struct {
	id          string
	orderType   OrderType
	side        Side
	timestamp   time.Time
	quantity    decimal.Decimal
	originalQty decimal.Decimal
	price       decimal.Decimal
	canceled    bool
	role        Role
	stop        decimal.Decimal
	tif         TIF
	oco         string
}

// NewMarketOrder creates new constant object Order
func NewMarketOrder(orderID string, side Side, quantity decimal.Decimal) *Order {
	return &Order{
		id:          orderID,
		orderType:   TypeMarket,
		side:        side,
		quantity:    quantity,
		originalQty: quantity.Copy(),
		price:       decimal.Zero,
		timestamp:   time.Now().UTC(),
		canceled:    false,
	}
}

// NewLimitOrder creates new constant object Order
func NewLimitOrder(orderID string, side Side, quantity, price decimal.Decimal, tif TIF, oco string) *Order {
	return &Order{
		id:          orderID,
		orderType:   TypeLimit,
		side:        side,
		quantity:    quantity,
		originalQty: quantity.Copy(),
		price:       price,
		timestamp:   time.Now().UTC(),
		canceled:    false,
		oco:         oco,
		tif:         tif,
	}
}

// NewStopLimitOrder creates new constant object Order
func NewStopLimitOrder(orderID string, side Side, quantity, price, stop decimal.Decimal, oco string) *Order {
	return &Order{
		id:          orderID,
		orderType:   TypeStopLimit,
		side:        side,
		quantity:    quantity,
		originalQty: quantity.Copy(),
		price:       price,
		timestamp:   time.Now().UTC(),
		canceled:    false,
		stop:        stop,
		oco:         oco,
	}
}

// ID returns OrderID field copy
func (o *Order) ID() string {
	return o.id
}

// Side returns side of the Order
func (o *Order) Side() Side {
	return o.side
}

// Quantity returns Quantity field copy
func (o *Order) Quantity() decimal.Decimal {
	return o.quantity
}

// OriginalQty returns originalQty field copy
func (o *Order) OriginalQty() decimal.Decimal {
	return o.originalQty
}

// SetQuantity set Quantity field
func (o *Order) SetQuantity(quantity decimal.Decimal) {
	o.quantity = quantity
}

// Price returns Price field copy
func (o *Order) Price() decimal.Decimal {
	return o.price
}

// StopPrice returns Price field copy
func (o *Order) StopPrice() decimal.Decimal {
	return o.stop
}

// Time returns timestamp field copy
func (o *Order) Time() time.Time {
	return o.timestamp
}

// OCO returns reference ID
func (o *Order) OCO() string {
	return o.oco
}

// TIF returns tif field
func (o *Order) TIF() TIF {
	return o.tif
}

// IsCanceled returns Canceled status
func (o *Order) IsCanceled() bool {
	return o.canceled
}

// Cancel set Canceled status
func (o *Order) Cancel() bool {
	o.canceled = true
	return o.canceled
}

// IsMarketOrder returns true if Order is MARKET
func (o *Order) IsMarketOrder() bool {
	return o.orderType == TypeMarket
}

// IsLimitOrder returns true if Order is LIMIT
func (o *Order) IsLimitOrder() bool {
	return o.orderType == TypeLimit
}

// IsStopOrder returns true if Order is STOP-LIMIT
func (o *Order) IsStopOrder() bool {
	return o.orderType == TypeStopLimit
}

// ActivateStopOrder transforms Stop-Order into Order
func (o *Order) ActivateStopOrder() {

	if !o.IsStopOrder() {
		panic("Order isn't Stop")
	}

	o.stop = decimal.Zero

	o.orderType = TypeLimit
}

// SetMaker sets Maker role
func (o *Order) SetMaker() {
	o.role = MAKER
}

// Role returns role of Order
func (o *Order) Role() Role {
	if o.role == MAKER {
		return MAKER
	}

	return TAKER
}

// String implements Stringer interface
func (o *Order) String() string {
	return fmt.Sprintf("\n\"%s\":\n\ttype: %s\n\tside: %s\n\tquantity: %s\n\toriginalQty: %s\n\tprice: %s\n\ttime: %s\n\tcanceled: %t\n\trole: %s\n", o.ID(), o.orderType, o.Side(), o.Quantity(), o.OriginalQty(), o.Price(), o.Time(), o.IsCanceled(), o.Role())
}
