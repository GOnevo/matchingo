package matchingo

// OrderType of the Order
type OrderType string

// Different order types
const (
	TypeMarket     OrderType = "MARKET"
	TypeLimit      OrderType = "LIMIT"
	TypeStopLimit  OrderType = "STOP-LIMIT"
	TypeStopMarket OrderType = "STOP-MARKET"
)

// STPMode selects what happens when an incoming order would trade against a
// resting order of the same participant. The incoming order's mode applies.
type STPMode uint8

// Self-trade prevention modes
const (
	STPNone         STPMode = iota // no prevention (default)
	STPCancelNewest                // cancel the incoming order's remainder
	STPCancelOldest                // cancel the resting order and keep matching
	STPCancelBoth                  // cancel the resting order and the incoming remainder
	STPDecrement                   // reduce both by the overlap without a trade
)

// Role of the Order
type Role string

// Different order roles
const (
	MAKER Role = "MAKER"
	TAKER Role = "TAKER"
)

// Side of the Order
type Side int

// Sell (asks) or Buy (bids)
const (
	Sell Side = iota
	Buy
)

// String implements fmt.Stringer interface
func (s Side) String() string {
	if s == Buy {
		return "BUY"
	}

	return "SELL"
}

// TIF of the Order
type TIF string

// Different order TIF
const (
	GTC TIF = "GTC"
	FOK TIF = "FOK"
	IOC TIF = "IOC"
	GTD TIF = "GTD"
)
