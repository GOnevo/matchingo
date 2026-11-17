// Example 06 — OCO (One-Cancels-Other) Orders
//
// An OCO pair links two orders by their IDs: when one is filled (or its
// stop activates and fills), the other is automatically canceled.
//
// Important: each order's OCO field must be set to the sibling's ORDER ID —
// the linkage is bidirectional and ID-based, not tag-based.
//
//	order-A  oco="order-B"
//	order-B  oco="order-A"
//
// Typical use-case — take-profit + stop-loss on a resting position:
//   - limit sell at $110 (take-profit: sell high)
//   - stop-limit sell at $90→$89 (stop-loss: cut losses if price falls)
//
// When either side executes, the engine cancels the other.
//
// Concepts shown:
//   - Bidirectional OCO linkage via order ID
//   - Done.Canceled lists the sibling that was auto-canceled
//   - New*Order oco parameter
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// ------------------------------------------------------------------ //
	// Scenario A: take-profit fills → stop-loss sibling is canceled.     //
	// Both orders reference each other's ID.                              //
	// ------------------------------------------------------------------ //
	ob.Process(matchingo.NewLimitOrder(
		"tp-sell", matchingo.Sell,
		matchingo.FromInt(5), matchingo.FromInt(110),
		matchingo.GTC, "sl-sell", // oco = sibling's order ID
	))
	ob.Process(matchingo.NewStopLimitOrder(
		"sl-sell", matchingo.Sell,
		matchingo.FromInt(5), matchingo.FromInt(89), matchingo.FromInt(90),
		"tp-sell", // oco = sibling's order ID
	))
	fmt.Println("OCO pair resting (take-profit on ask, stop-loss in stop book):")
	fmt.Println(ob)

	// A buyer at $110 fills the take-profit → engine auto-cancels sl-sell.
	done, _ := ob.Process(matchingo.NewMarketOrder("mkt-buy", matchingo.Buy, matchingo.FromInt(5)))
	fmt.Printf("\nMarket buy fills take-profit:\n")
	fmt.Printf("  processed=%s  canceled=%v\n", done.Processed, done.Canceled)
	// canceled=["sl-sell"]
	fmt.Printf("  sl-sell still in book? %v\n", ob.GetOrder("sl-sell") != nil)
	fmt.Println(ob) // empty — both orders gone

	// ------------------------------------------------------------------ //
	// Scenario B: stop-loss activates and fills → take-profit is canceled.//
	// ------------------------------------------------------------------ //
	ob2 := matchingo.NewOrderBook()
	ob2.Process(matchingo.NewLimitOrder(
		"tp2-sell", matchingo.Sell,
		matchingo.FromInt(5), matchingo.FromInt(110),
		matchingo.GTC, "sl2-sell",
	))
	ob2.Process(matchingo.NewStopLimitOrder(
		"sl2-sell", matchingo.Sell,
		matchingo.FromInt(5), matchingo.FromInt(89), matchingo.FromInt(90),
		"tp2-sell",
	))

	// Seed a buyer at $89 so the activated stop-sell has something to hit.
	ob2.Process(matchingo.NewLimitOrder("bid-below", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(89), matchingo.GTC, ""))

	// Trade at $90 triggers sl2-sell → it activates, matches bid-below,
	// and the engine cancels tp2-sell.
	ob2.Process(matchingo.NewLimitOrder("ask-trig", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(90), matchingo.GTC, ""))
	trig, _ := ob2.Process(matchingo.NewLimitOrder("buy-trig", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(90), matchingo.GTC, ""))

	fmt.Printf("\nTrade at $90 triggers stop-loss:\n")
	fmt.Printf("  trig activated=%v\n", trig.Activated)
	for _, act := range trig.Activations {
		fmt.Printf("  activation %s → processed=%s  canceled=%v\n",
			act.Order.ID(), act.Processed, act.Canceled)
	}
	fmt.Printf("  tp2-sell still in book? %v\n", ob2.GetOrder("tp2-sell") != nil)
	fmt.Println(ob2) // both orders consumed/canceled
}
