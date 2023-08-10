package matchingo

import "errors"

// OrderBook errors
var (
	ErrInvalidQuantity      = errors.New("orderbook: invalid GetOrder Quantity")
	ErrInvalidPrice         = errors.New("orderbook: invalid GetOrder Price")
	ErrInvalidTif           = errors.New("orderbook: invalid GetOrder time in force")
	ErrOrderExists          = errors.New("orderbook: GetOrder already exists")
	ErrInsufficientQuantity = errors.New("orderbook: insufficient Volume to calculate Price")
)
