package matchingo

// OrderType of the Order
type OrderType string

const (
	TypeMarket    OrderType = "MARKET"
	TypeLimit     OrderType = "LIMIT"
	TypeStopLimit OrderType = "STOP-LIMIT"
)

// Role of the Order
type Role string

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

const (
	GTC TIF = "GTC"
	FOK TIF = "FOK"
	IOC TIF = "IOC"
)
