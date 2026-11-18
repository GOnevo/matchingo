// Example 14 — Stop-Market Orders
//
// A stop-market order is inactive until the market trades through its stop
// price, then it becomes a MARKET order: no limit price, it takes whatever
// liquidity is available. Any quantity the book cannot absorb is canceled.
//
// Concepts shown:
//   - matchingo.NewStopMarketOrder(id, side, qty, stopPrice, oco)
//   - Activation sweeps multiple price levels (no price protection)
//   - Unfilled remainder is canceled (done.Canceled), never rests
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// "If price rises to $100, buy 4 at any price."
	ob.Process(matchingo.NewStopMarketOrder("stop-buy", matchingo.Buy, matchingo.FromInt(4), matchingo.FromInt(100), ""))

	ob.Process(matchingo.NewLimitOrder("ask-100", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(100), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-105", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(105), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-110", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(110), matchingo.GTC, ""))

	// A trade at $100 triggers the stop.
	done, _ := ob.Process(matchingo.NewMarketOrder("trigger", matchingo.Buy, matchingo.FromInt(1)))
	fmt.Printf("activated=%v\n", done.Activated)
	for _, act := range done.Activations {
		fmt.Printf("  %s → processed=%s left=%s\n", act.Order.ID(), act.Processed, act.Left)
		for _, tr := range act.Trades[1:] {
			fmt.Printf("    filled %s @ %s against %s\n", tr.Quantity, tr.Price, tr.OrderID)
		}
	}

	// Thin book: a sell stop asks for 5 but only 2 can be filled.
	ob2 := matchingo.NewOrderBook()
	ob2.Process(matchingo.NewStopMarketOrder("stop-sell", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(90), ""))
	ob2.Process(matchingo.NewLimitOrder("bid-90", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(90), matchingo.GTC, ""))
	ob2.Process(matchingo.NewLimitOrder("bid-89", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(89), matchingo.GTC, ""))
	done2, _ := ob2.Process(matchingo.NewMarketOrder("trigger2", matchingo.Sell, matchingo.FromInt(1)))
	act := done2.Activations[0]
	fmt.Printf("\nthin book: processed=%s left=%s canceled=%v\n", act.Processed, act.Left, act.Canceled)
}
