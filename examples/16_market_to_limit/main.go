// Example 16 — Market-to-Limit (MTL) Orders
//
// An MTL order starts as a market order. If the book runs out of liquidity
// before it is filled, the remainder is not canceled: it rests as a limit
// order at the price of the last trade. If nothing fills, it is canceled.
//
// Concepts shown:
//   - matchingo.NewMarketToLimitOrder(id, side, qty)
//   - done.Stored is true when a remainder was placed on the book
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()
	ob.Process(matchingo.NewLimitOrder("ask-10", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-12", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(12), matchingo.GTC, ""))

	done, _ := ob.Process(matchingo.NewMarketToLimitOrder("mtl", matchingo.Buy, matchingo.FromInt(5)))
	fmt.Printf("processed=%s left=%s stored=%v\n", done.Processed, done.Left, done.Stored)
	fmt.Println(ob) // remainder rests as a bid at 12
}
