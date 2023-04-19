package matchingo

import "errors"

// OrderBook errors
var (
	ErrInvalidQuantity      = errors.New("orderbook: invalid Order Quantity")
	ErrInvalidPrice         = errors.New("orderbook: invalid Order Price")
	ErrInvalidTif           = errors.New("orderbook: invalid Order time in force")
	ErrOrderExists          = errors.New("orderbook: Order already exists")
	ErrInsufficientQuantity = errors.New("orderbook: insufficient Volume to calculate Price")
)
