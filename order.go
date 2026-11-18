package matchingo

import (
	"encoding/json"
	"time"
)

// TradeOrder structure
type TradeOrder struct {
	OrderID  string
	Role     Role
	Price    Decimal
	IsQuote  bool
	Quantity Decimal
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

func newTradeOrder(order *Order, quantity, price Decimal) TradeOrder {
	return TradeOrder{
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
	quantity    Decimal
	originalQty Decimal
	price       Decimal
	canceled    bool
	role        Role
	stop        Decimal
	tif         TIF
	oco         string
	peakQty     Decimal
	hiddenQty   Decimal
	postOnly    bool
	ext         *orderExt // rarely used attributes; nil for plain orders
	createdAt   time.Time
}

// orderExt holds attributes few orders use, keeping the common Order small.
type orderExt struct {
	trail       Decimal
	trailPct    bool
	toLimit     bool
	expiresAt   time.Time
	participant string
	stp         STPMode
	ttl         time.Duration
}

var noExt orderExt

// extra returns the order's extension for reading (a shared zero value if none).
func (o *Order) extra() *orderExt {
	if o.ext == nil {
		return &noExt
	}
	return o.ext
}

// mut returns the order's extension for writing, allocating it on first use.
func (o *Order) mut() *orderExt {
	if o.ext == nil {
		o.ext = &orderExt{}
	}
	return o.ext
}

// NewMarketOrder creates new constant object Order
func NewMarketOrder(orderID string, side Side, quantity Decimal) *Order {

	if quantity.LessThanOrEqual(Zero) {
		panic(ErrInvalidQuantity)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeMarket,
		side:        side,
		quantity:    quantity,
		originalQty: quantity,
		price:       Zero,
		canceled:    false,
	}
}

// NewMarketToLimitOrder creates a market order whose unfilled remainder rests as a
// limit order at the last trade price instead of being canceled.
func NewMarketToLimitOrder(orderID string, side Side, quantity Decimal) *Order {
	o := NewMarketOrder(orderID, side, quantity)
	o.mut().toLimit = true
	return o
}

// IsMarketToLimit returns true if the order converts to a limit order after a partial fill
func (o *Order) IsMarketToLimit() bool {
	return o.extra().toLimit
}

// NewMarketQuoteOrder creates new constant object Order, but quantity is in Quote mode
func NewMarketQuoteOrder(orderID string, side Side, quantity Decimal) *Order {

	if quantity.LessThanOrEqual(Zero) {
		panic(ErrInvalidQuantity)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeMarket,
		side:        side,
		quantity:    quantity,
		originalQty: quantity,
		price:       Zero,
		canceled:    false,
		isQuote:     true,
	}
}

// NewLimitOrder creates new constant object Order
func NewLimitOrder(orderID string, side Side, quantity, price Decimal, tif TIF, oco string) *Order {

	if quantity.LessThanOrEqual(Zero) {
		panic(ErrInvalidQuantity)
	}

	if price.LessThanOrEqual(Zero) {
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
func NewStopLimitOrder(orderID string, side Side, quantity, price, stop Decimal, oco string) *Order {

	if quantity.LessThanOrEqual(Zero) {
		panic(ErrInvalidQuantity)
	}

	if price.LessThanOrEqual(Zero) || stop.LessThanOrEqual(Zero) {
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

// NewStopMarketOrder creates a stop order that activates as a MARKET order
func NewStopMarketOrder(orderID string, side Side, quantity, stop Decimal, oco string) *Order {

	if quantity.LessThanOrEqual(Zero) {
		panic(ErrInvalidQuantity)
	}

	if stop.LessThanOrEqual(Zero) {
		panic(ErrInvalidPrice)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeStopMarket,
		side:        side,
		quantity:    quantity,
		originalQty: quantity,
		price:       Zero,
		canceled:    false,
		stop:        stop,
		oco:         oco,
	}
}

// NewTrailingStopOrder creates a stop-market order whose stop price follows the
// trade price at a fixed distance. stop is the initial stop price; trail is the offset.
func NewTrailingStopOrder(orderID string, side Side, quantity, stop, trail Decimal, oco string) *Order {
	if trail.LessThanOrEqual(Zero) {
		panic(ErrInvalidPrice)
	}
	o := NewStopMarketOrder(orderID, side, quantity, stop, oco)
	o.mut().trail = trail
	return o
}

// NewTrailingStopPercentOrder is like NewTrailingStopOrder but trail is a percentage
// of the trade price (5 means 5%). It must be below 100.
func NewTrailingStopPercentOrder(orderID string, side Side, quantity, stop, percent Decimal, oco string) *Order {
	if percent.LessThanOrEqual(Zero) || percent.GreaterThanOrEqual(FromInt(100)) {
		panic(ErrInvalidPrice)
	}
	o := NewStopMarketOrder(orderID, side, quantity, stop, oco)
	o.mut().trail = percent
	o.mut().trailPct = true
	return o
}

// IsTrailing returns true if the stop price follows the market
func (o *Order) IsTrailing() bool {
	return o.extra().trail.GreaterThan(Zero)
}

// TrailValue returns the trailing offset (a percentage when TrailIsPercent is true)
func (o *Order) TrailValue() Decimal {
	return o.extra().trail
}

// TrailIsPercent returns true if the trailing offset is a percentage
func (o *Order) TrailIsPercent() bool {
	return o.extra().trailPct
}

// NewPostOnlyOrder creates a new post-only LIMIT order
func NewPostOnlyOrder(orderID string, side Side, qty, price Decimal, tif TIF, oco string) *Order {

	if qty.LessThanOrEqual(Zero) {
		panic(ErrInvalidQuantity)
	}

	if price.LessThanOrEqual(Zero) {
		panic(ErrInvalidPrice)
	}

	if tif != "" && tif != GTC && tif != FOK && tif != IOC {
		panic(ErrInvalidTif)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeLimit,
		side:        side,
		quantity:    qty,
		originalQty: qty,
		price:       price,
		canceled:    false,
		oco:         oco,
		tif:         tif,
		postOnly:    true,
	}
}

// NewGTDOrder creates a good-till-date LIMIT order that is canceled once expiresAt passes
func NewGTDOrder(orderID string, side Side, quantity, price Decimal, expiresAt time.Time, oco string) *Order {
	if expiresAt.IsZero() {
		panic(ErrInvalidTif)
	}
	o := NewLimitOrder(orderID, side, quantity, price, GTC, oco)
	o.tif = GTD
	o.mut().expiresAt = expiresAt
	return o
}

// SetTTL gives a not-yet-processed order a lifetime: when the book accepts it, the
// order becomes GTD and expires ttl after the book clock's current time. It works on
// limit and stop orders with TIF GTC (or unset); Process rejects other TIFs with ErrInvalidTif.
func (o *Order) SetTTL(ttl time.Duration) {
	if ttl <= 0 {
		panic(ErrInvalidTif)
	}
	o.mut().ttl = ttl
}

// ExpiresAt returns the expiry time of a GTD order (zero for other orders)
func (o *Order) ExpiresAt() time.Time {
	return o.extra().expiresAt
}

// SetSTP enables self-trade prevention: when this order would trade against a resting
// order with the same participant ID, mode decides what is canceled or decremented.
func (o *Order) SetSTP(participant string, mode STPMode) {
	if mode > STPDecrement {
		panic(ErrInvalidSTP)
	}
	o.mut().participant = participant
	o.mut().stp = mode
}

// Participant returns the participant ID used for self-trade prevention
func (o *Order) Participant() string {
	return o.extra().participant
}

// STPMode returns the self-trade prevention mode
func (o *Order) STPMode() STPMode {
	return o.extra().stp
}

// IsPostOnly returns true if this is a post-only order
func (o *Order) IsPostOnly() bool {
	return o.postOnly
}

// NewIcebergOrder creates a new iceberg LIMIT order
func NewIcebergOrder(orderID string, side Side, totalQty, peakQty, price Decimal, tif TIF, oco string) *Order {

	if totalQty.LessThanOrEqual(Zero) {
		panic(ErrInvalidQuantity)
	}

	if price.LessThanOrEqual(Zero) {
		panic(ErrInvalidPrice)
	}

	if peakQty.GreaterThanOrEqual(totalQty) {
		panic(ErrInvalidQuantity)
	}

	if tif != "" && tif != GTC && tif != FOK && tif != IOC {
		panic(ErrInvalidTif)
	}

	return &Order{
		id:          orderID,
		orderType:   TypeLimit,
		side:        side,
		quantity:    peakQty,
		originalQty: totalQty,
		price:       price,
		canceled:    false,
		oco:         oco,
		tif:         tif,
		peakQty:     peakQty,
		hiddenQty:   totalQty.Sub(peakQty),
	}
}

// IsIceberg returns true if this is an iceberg order
func (o *Order) IsIceberg() bool {
	return o.peakQty.GreaterThan(Zero)
}

// PeakQty returns the peak quantity of an iceberg order
func (o *Order) PeakQty() Decimal {
	return o.peakQty
}

// HiddenQty returns the remaining hidden quantity of an iceberg order
func (o *Order) HiddenQty() Decimal {
	return o.hiddenQty
}

// ID returns OrderID field copy
func (o *Order) ID() string {
	return o.id
}

// CreatedAt returns the time the order was created.
func (o *Order) CreatedAt() time.Time {
	return o.createdAt
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
func (o *Order) Quantity() Decimal {
	return o.quantity
}

// OriginalQty returns originalQty field copy
func (o *Order) OriginalQty() Decimal {
	return o.originalQty
}

// SetQuantity set Quantity field
func (o *Order) SetQuantity(quantity Decimal) {
	o.quantity = quantity
}

// DecreaseQuantity set Quantity field
func (o *Order) DecreaseQuantity(quantity Decimal) {
	o.quantity = o.quantity.Sub(quantity)
}

// Price returns Price field copy
func (o *Order) Price() Decimal {
	return o.price
}

// StopPrice returns Price field copy
func (o *Order) StopPrice() Decimal {
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

// SetTIF sets the time-in-force for stop-limit orders before they activate.
func (o *Order) SetTIF(tif TIF) {
	o.tif = tif
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

// IsStopOrder returns true if Order is STOP-LIMIT or STOP-MARKET
func (o *Order) IsStopOrder() bool {
	return o.orderType == TypeStopLimit || o.orderType == TypeStopMarket
}

// ActivateStopOrder transforms Stop-GetOrder into Order
func (o *Order) ActivateStopOrder() {

	if !o.IsStopOrder() {
		panic("GetOrder isn't Stop")
	}

	o.stop = Zero
	if o.ext != nil {
		o.ext.trail = Zero
	}

	if o.orderType == TypeStopMarket {
		o.orderType = TypeMarket
		return
	}

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
