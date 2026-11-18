// Example 15 — Trailing Stop Orders
//
// A trailing stop is a stop-market order whose stop price follows the trade
// price at a fixed offset (or percentage). It only moves in the favorable
// direction: a sell stop ratchets up, a buy stop ratchets down.
//
// Concepts shown:
//   - matchingo.NewTrailingStopOrder(id, side, qty, initialStop, offset, oco)
//   - matchingo.NewTrailingStopPercentOrder(id, side, qty, initialStop, percent, oco)
//   - The stop is re-evaluated on every trade, then activates as a market order
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func trade(ob *matchingo.OrderBook, id string, price int) *matchingo.Done {
	ob.Process(matchingo.NewLimitOrder(id+"-ask", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(price), matchingo.GTC, ""))
	done, _ := ob.Process(matchingo.NewLimitOrder(id+"-bid", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(price), matchingo.GTC, ""))
	return done
}

func main() {
	ob := matchingo.NewOrderBook()

	// Sell 1 if price falls $5 below its peak; starts with stop at $95.
	ob.Process(matchingo.NewTrailingStopOrder("ts", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(95), matchingo.FromInt(5), ""))

	for i, p := range []int{100, 110, 108, 120} {
		trade(ob, fmt.Sprintf("t%d", i), p)
		fmt.Printf("trade @ %d → stop = %s\n", p, ob.GetOrder("ts").StopPrice())
	}

	// Liquidity for the stop to sell into, then a trade through the stop ($115).
	ob.Process(matchingo.NewLimitOrder("bid-exit", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), matchingo.GTC, ""))
	done := trade(ob, "drop", 114)
	fmt.Printf("trade @ 114 → activated=%v processed=%s\n", done.Activated, done.Activations[0].Processed)

	// Percentage variant: trail 10% below the peak.
	ob2 := matchingo.NewOrderBook()
	ob2.Process(matchingo.NewTrailingStopPercentOrder("pct", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(50), matchingo.FromInt(10), ""))
	trade(ob2, "p", 200)
	fmt.Printf("percent trail after trade @ 200 → stop = %s\n", ob2.GetOrder("pct").StopPrice())
}
