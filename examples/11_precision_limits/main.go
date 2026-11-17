package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Allow at most 2 decimal places on quantities, 4 on prices.
	ob.SetQuantityPrecision(2)
	ob.SetPricePrecision(4)

	// Valid order: 1.25 qty @ 10.1234 price — accepted.
	done, err := ob.Process(matchingo.NewLimitOrder(
		"sell1", matchingo.Sell,
		matchingo.FromFloat(1.25), matchingo.FromFloat(10.1234),
		"", "",
	))
	fmt.Printf("sell1 stored=%v err=%v\n", done.Stored, err)

	// Invalid quantity: 1.251 has 3 decimal places → ErrQuantityPrecision.
	_, err = ob.Process(matchingo.NewLimitOrder(
		"buy1", matchingo.Buy,
		matchingo.FromFloat(1.251), matchingo.FromFloat(10.1234),
		"", "",
	))
	fmt.Printf("buy1 (bad qty)  err=%v\n", err)

	// Invalid price: 10.12345 has 5 decimal places → ErrPricePrecision.
	_, err = ob.Process(matchingo.NewLimitOrder(
		"buy2", matchingo.Buy,
		matchingo.FromFloat(1.25), matchingo.FromFloat(10.12345),
		"", "",
	))
	fmt.Printf("buy2 (bad price) err=%v\n", err)

	// Valid buy: matches the resting sell1.
	done, err = ob.Process(matchingo.NewLimitOrder(
		"buy3", matchingo.Buy,
		matchingo.FromFloat(1.25), matchingo.FromFloat(10.1234),
		"", "",
	))
	fmt.Printf("buy3 processed=%.2f err=%v\n", done.Processed.Float64(), err)
}
