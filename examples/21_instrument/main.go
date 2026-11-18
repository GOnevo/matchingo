// Example 21 — Instrument Metadata
//
// WithInstrument validates every order against the symbol's constraints before
// it reaches the book. A zero field means "unchecked".
//
//   - TickSize    — limit and stop prices must be a multiple of it
//   - LotSize     — quantities must be a multiple of it
//   - Min/MaxQuantity — total order size range (total size for icebergs)
//   - Min/MaxPrice    — static price band
//
// Violations return ErrTickSize, ErrLotSize, ErrQuantityRange or ErrPriceBand
// (wrapped with details; use errors.Is). Modify, ModifyStop and Replace are
// checked too. Quote-quantity market orders skip lot/quantity checks.
package main

import (
	"errors"
	"fmt"

	"github.com/gonevo/matchingo"
)

func dec(s string) matchingo.Decimal {
	d, _ := matchingo.FromString(s)
	return d
}

func main() {
	ob := matchingo.NewOrderBook(matchingo.WithInstrument(matchingo.Instrument{
		Symbol:      "BTC-USD",
		TickSize:    dec("0.5"),
		LotSize:     dec("0.1"),
		MinQuantity: dec("0.2"),
		MaxQuantity: dec("10"),
		MinPrice:    dec("50"),
		MaxPrice:    dec("200"),
	}))

	try := func(label string, o *matchingo.Order) {
		_, err := ob.Process(o)
		fmt.Printf("%-18s ok=%-5v tick=%-5v lot=%-5v range=%-5v band=%-5v\n", label, err == nil,
			errors.Is(err, matchingo.ErrTickSize), errors.Is(err, matchingo.ErrLotSize),
			errors.Is(err, matchingo.ErrQuantityRange), errors.Is(err, matchingo.ErrPriceBand))
	}
	try("valid", matchingo.NewLimitOrder("a", matchingo.Buy, dec("1.5"), dec("100.5"), matchingo.GTC, ""))
	try("bad tick", matchingo.NewLimitOrder("b", matchingo.Buy, dec("1"), dec("100.2"), matchingo.GTC, ""))
	try("bad lot", matchingo.NewLimitOrder("c", matchingo.Buy, dec("1.05"), dec("100"), matchingo.GTC, ""))
	try("too small", matchingo.NewLimitOrder("d", matchingo.Buy, dec("0.1"), dec("100"), matchingo.GTC, ""))
	try("outside band", matchingo.NewLimitOrder("e", matchingo.Buy, dec("1"), dec("500"), matchingo.GTC, ""))
}
