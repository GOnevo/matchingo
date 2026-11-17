package matchingo

import (
	"errors"
	"fmt"
	"strconv"
)

// OrderBook errors
var (
	ErrInvalidOrder         = errors.New("orderbook: invalid order")
	ErrInvalidSide          = errors.New("orderbook: invalid order side")
	ErrCanceledOrder        = errors.New("orderbook: order is canceled")
	ErrInvalidQuantity      = errors.New("orderbook: invalid GetOrder Quantity")
	ErrInvalidPrice         = errors.New("orderbook: invalid GetOrder Price")
	ErrInvalidTif           = errors.New("orderbook: invalid GetOrder time in force")
	ErrOrderExists          = errors.New("orderbook: GetOrder already exists")
	ErrInsufficientQuantity = errors.New("orderbook: insufficient Volume to calculate Price")
	ErrQuantityPrecision    = errors.New("orderbook: quantity exceeds allowed precision")
	ErrPricePrecision       = errors.New("orderbook: price exceeds allowed precision")
	ErrOrderNotFound        = errors.New("orderbook: order not found")
	ErrNotStopOrder         = errors.New("orderbook: order is not a stop-limit order")
	ErrNotLimitOrder        = errors.New("orderbook: order is not a limit order")
)

type precisionError struct {
	sentinel  error
	value     string
	maxDigits int
}

func (e *precisionError) Error() string {
	return e.sentinel.Error() + ": got " + e.value + ", max " + strconv.Itoa(e.maxDigits) + " decimal places"
}
func (e *precisionError) Unwrap() error { return e.sentinel }

func errQuantityPrecision(qty Decimal, maxDecimals int) error {
	return &precisionError{ErrQuantityPrecision, qty.String(), maxDecimals}
}

func errPricePrecision(price Decimal, maxDecimals int) error {
	return &precisionError{ErrPricePrecision, price.String(), maxDecimals}
}

// keep fmt imported for D() helper in decimal.go
var _ = fmt.Sprintf
