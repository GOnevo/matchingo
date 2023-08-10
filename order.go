package matchingo

import (
	"encoding/json"

	"github.com/nikolaydubina/fpdecimal"
)

// TradeOrder structure
type TradeOrder struct {
	OrderID  string
	Role     Role
	Price    fpdecimal.Decimal
	IsQuote  bool
	Quantity fpdecimal.Decimal
}

// MarshalJSON implements Marshaler interface
func (t *TradeOrder) MarshalJSON() ([]byte, error) {
	customStruct := struct {
		OrderID  string `json:"orderID"`
		Role     Role   `json:"role"`
		IsQuote  bool   `json:"isQuote"`
		Price    string `json:"price"`
		Quantity string `json:"quantity"`
	}{
		OrderID:  t.OrderID,
		Role:     t.Role,
		IsQuote:  t.IsQuote,
		Price:    t.Price.String(),
		Quantity: t.Quantity.String(),
	}
	return json.Marshal(customStruct)
}

func newTradeOrder(order *Order, quantity, price fpdecimal.Decimal) *TradeOrder {
	return &TradeOrder{
		OrderID:  order.ID(),
		Role:     order.Role(),
		Price:    price,
		IsQuote:  order.IsQuote(),
		Quantity: quantity,
	}
}

// Order stores information about order
type Order struct {
	id          string
	orderType   OrderType
	side        Side
	isQuote     bool
	quantity    fpdecimal.Decimal
	originalQty fpdecimal.Decimal
	price       fpdecimal.Decimal
	canceled    bool
	role        Role
	stop        fpdecimal.Decimal
	tif         TIF
	oco         string
}

// NewMarketOrder creates new constant object Order
func NewMarketOrder(orderID string, side Side, quantity fpdecimal.Decimal) *Order {

	if quantity.LessThanOrEqual(fpdecimal.Zero) {
		panic(ErrInvalidQuantity)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeMarket,
		side:        side,
		quantity:    quantity,
		originalQty: quantity,
		price:       fpdecimal.Zero,
		canceled:    false,
	}
}

// NewMarketQuoteOrder creates new constant object Order, but quantity is in Quote mode
func NewMarketQuoteOrder(orderID string, side Side, quantity fpdecimal.Decimal) *Order {

	if quantity.LessThanOrEqual(fpdecimal.Zero) {
		panic(ErrInvalidQuantity)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeMarket,
		side:        side,
		quantity:    quantity,
		originalQty: quantity,
		price:       fpdecimal.Zero,
		canceled:    false,
		isQuote:     true,
	}
}

// NewLimitOrder creates new constant object Order
func NewLimitOrder(orderID string, side Side, quantity, price fpdecimal.Decimal, tif TIF, oco string) *Order {

	if quantity.LessThanOrEqual(fpdecimal.Zero) {
		panic(ErrInvalidQuantity)
	}

	if price.LessThanOrEqual(fpdecimal.Zero) {
		panic(ErrInvalidPrice)
	}

	if tif != "" && tif != GTC && tif != FOK && tif != IOC {
		panic(ErrInvalidTif)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeLimit,
		side:        side,
		quantity:    quantity,
		originalQty: quantity,
		price:       price,
		canceled:    false,
		oco:         oco,
		tif:         tif,
	}
}

// NewStopLimitOrder creates new constant object Order
func NewStopLimitOrder(orderID string, side Side, quantity, price, stop fpdecimal.Decimal, oco string) *Order {

	if quantity.LessThanOrEqual(fpdecimal.Zero) {
		panic(ErrInvalidQuantity)
	}

	if price.LessThanOrEqual(fpdecimal.Zero) || stop.LessThanOrEqual(fpdecimal.Zero) {
		panic(ErrInvalidPrice)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeStopLimit,
		side:        side,
		quantity:    quantity,
		originalQty: quantity,
		price:       price,
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

// IsQuote returns isQuote field copy
func (o *Order) IsQuote() bool {
	return o.isQuote
}

// Quantity returns Quantity field copy
func (o *Order) Quantity() fpdecimal.Decimal {
	return o.quantity
}

// OriginalQty returns originalQty field copy
func (o *Order) OriginalQty() fpdecimal.Decimal {
	return o.originalQty
}

// SetQuantity set Quantity field
func (o *Order) SetQuantity(quantity fpdecimal.Decimal) {
	o.quantity = quantity
}

// DecreaseQuantity set Quantity field
func (o *Order) DecreaseQuantity(quantity fpdecimal.Decimal) {
	o.quantity = o.quantity.Sub(quantity)
}

// Price returns Price field copy
func (o *Order) Price() fpdecimal.Decimal {
	return o.price
}

// StopPrice returns Price field copy
func (o *Order) StopPrice() fpdecimal.Decimal {
	return o.stop
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

// ActivateStopOrder transforms Stop-GetOrder into Order
func (o *Order) ActivateStopOrder() {

	if !o.IsStopOrder() {
		panic("GetOrder isn't Stop")
	}

	o.stop = fpdecimal.Zero

	o.orderType = TypeLimit
}

// SetMaker sets Maker role
func (o *Order) SetMaker() {
	o.role = MAKER
}

// SetTaker sets Taker role
func (o *Order) SetTaker() {
	o.role = TAKER
}

// Role returns role of Order
func (o *Order) Role() Role {
	if o.role == MAKER {
		return MAKER
	}

	return TAKER
}

// ToSimple returns TradeOrder
func (o *Order) ToSimple() *TradeOrder {
	return &TradeOrder{
		OrderID:  o.ID(),
		Role:     o.Role(),
		IsQuote:  o.IsQuote(),
		Quantity: o.Quantity(),
		Price:    o.Price(),
	}
}

// String implements Stringer interface
func (o *Order) String() string {
	j, _ := o.ToSimple().MarshalJSON()
	return string(j)
}
