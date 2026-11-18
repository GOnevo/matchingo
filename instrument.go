package matchingo

import "fmt"

// Instrument describes the trading constraints of the symbol an OrderBook serves.
// A zero field means "unchecked". Orders that violate a constraint are rejected
// before they reach the book.
type Instrument struct {
	Symbol      string
	TickSize    Decimal // prices (limit and stop) must be a multiple of this
	LotSize     Decimal // quantities must be a multiple of this
	MinQuantity Decimal // smallest allowed order quantity (total, for icebergs)
	MaxQuantity Decimal // largest allowed order quantity (total, for icebergs)
	MinPrice    Decimal // lowest allowed limit/stop price
	MaxPrice    Decimal // highest allowed limit/stop price
}

type constraintError struct {
	sentinel error
	detail   string
}

func (e *constraintError) Error() string { return e.sentinel.Error() + ": " + e.detail }
func (e *constraintError) Unwrap() error { return e.sentinel }

func (in *Instrument) validate() {
	for _, d := range []Decimal{in.TickSize, in.LotSize, in.MinQuantity, in.MaxQuantity, in.MinPrice, in.MaxPrice} {
		if d.LessThan(Zero) {
			panic(ErrInvalidInstrument)
		}
	}
	if in.MaxQuantity.GreaterThan(Zero) && in.MinQuantity.GreaterThan(in.MaxQuantity) {
		panic(ErrInvalidInstrument)
	}
	if in.MaxPrice.GreaterThan(Zero) && in.MinPrice.GreaterThan(in.MaxPrice) {
		panic(ErrInvalidInstrument)
	}
}

func multipleOf(d, step Decimal) bool {
	return step.LessThanOrEqual(Zero) || d.Scaled()%step.Scaled() == 0
}

// checkQuantity validates total (min/max) and each visible/hidden slice (lot) of an order.
func (in *Instrument) checkQuantity(total Decimal, slices ...Decimal) error {
	for _, q := range slices {
		if !multipleOf(q, in.LotSize) {
			return &constraintError{ErrLotSize, fmt.Sprintf("quantity %s is not a multiple of %s", q, in.LotSize)}
		}
	}
	if total.LessThan(in.MinQuantity) || (in.MaxQuantity.GreaterThan(Zero) && total.GreaterThan(in.MaxQuantity)) {
		return &constraintError{ErrQuantityRange, fmt.Sprintf("quantity %s outside [%s, %s]", total, in.MinQuantity, in.MaxQuantity)}
	}
	return nil
}

func (in *Instrument) checkPrice(price Decimal) error {
	if !multipleOf(price, in.TickSize) {
		return &constraintError{ErrTickSize, fmt.Sprintf("price %s is not a multiple of %s", price, in.TickSize)}
	}
	if price.LessThan(in.MinPrice) || (in.MaxPrice.GreaterThan(Zero) && price.GreaterThan(in.MaxPrice)) {
		return &constraintError{ErrPriceBand, fmt.Sprintf("price %s outside [%s, %s]", price, in.MinPrice, in.MaxPrice)}
	}
	return nil
}

func (in *Instrument) checkOrder(o *Order) error {
	if !o.IsQuote() { // a quote quantity is a budget, not a number of lots
		slices := []Decimal{o.Quantity()}
		if o.IsIceberg() {
			slices = append(slices, o.HiddenQty())
		}
		if err := in.checkQuantity(o.OriginalQty(), slices...); err != nil {
			return err
		}
	}
	if !o.IsMarketOrder() && !o.IsStopOrder() || o.orderType == TypeStopLimit {
		if err := in.checkPrice(o.Price()); err != nil {
			return err
		}
	}
	if o.IsStopOrder() {
		return in.checkPrice(o.StopPrice())
	}
	return nil
}
