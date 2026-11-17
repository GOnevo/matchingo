// Example 08 — Post-Only Orders
//
// A post-only order is a limit order that guarantees maker status.
// If placing it would result in an immediate match (taker fill), it is
// rejected instead of executing.
//
// Exchanges use this to let market-makers earn the maker rebate without
// accidentally paying the taker fee.
//
// Concepts shown:
//   - matchingo.NewPostOnlyOrder
//   - A "would match" post-only is rejected (Done.Stored=false, no trades)
//   - A non-crossing post-only rests normally
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Seed a sell at $100.
	ob.Process(matchingo.NewLimitOrder("ask-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	fmt.Println("Book seeded with ask at $100:")
	fmt.Println(ob)

	// ------------------------------------------------------------------ //
	// Post-only buy at $100 — would cross the ask.                       //
	// The engine rejects it: no trade, not stored.                        //
	// ------------------------------------------------------------------ //
	done, _ := ob.Process(matchingo.NewPostOnlyOrder(
		"po-cross", matchingo.Buy,
		matchingo.FromInt(5), matchingo.FromInt(100),
		matchingo.GTC, "",
	))
	fmt.Printf("\nPost-only buy at $100 (would cross) → stored=%v  trades=%d\n",
		done.Stored, len(done.Trades))
	fmt.Println(ob) // ask-1 untouched

	// ------------------------------------------------------------------ //
	// Post-only buy at $99 — does not cross; rests as a maker order.     //
	// ------------------------------------------------------------------ //
	done, _ = ob.Process(matchingo.NewPostOnlyOrder(
		"po-rest", matchingo.Buy,
		matchingo.FromInt(5), matchingo.FromInt(99),
		matchingo.GTC, "",
	))
	fmt.Printf("\nPost-only buy at $99 (safe) → stored=%v  trades=%d\n",
		done.Stored, len(done.Trades))
	fmt.Println(ob) // bid at $99 and ask at $100 side by side
}
