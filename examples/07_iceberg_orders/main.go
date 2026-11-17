// Example 07 — Iceberg Orders
//
// An iceberg order hides most of its volume from the market.  Only the
// "peak" quantity is visible at any time; when the peak is fully consumed
// the next slice is automatically revealed until the hidden reserve is gone.
//
// This prevents large orders from moving the market against themselves.
//
// E.g. NewIcebergOrder("id", Sell, totalQty=100, peakQty=10, price=100, GTC, "")
//
//	→ shows 10 units; after those fill, another 10 appear, and so on.
//
// Concepts shown:
//   - matchingo.NewIcebergOrder
//   - Peak replenishment is automatic — no caller action needed
//   - The book depth reflects only the visible peak, not total volume
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Place an iceberg sell: total 50 units, but only 10 visible at a time.
	iceberg := matchingo.NewIcebergOrder(
		"ice-sell",
		matchingo.Sell,
		matchingo.FromInt(50), // total hidden + visible quantity
		matchingo.FromInt(10), // peak (visible slice)
		matchingo.FromInt(100),
		matchingo.GTC,
		"",
	)
	ob.Process(iceberg)

	fmt.Println("Iceberg posted (only peak is visible):")
	fmt.Println(ob)
	// Depth shows 10 units at $100, not 50.
	fmt.Printf("Depth: %s\n\n", ob.DepthJSON())

	// Buy 25 units in one shot.  The iceberg auto-replenishes twice:
	//   round 1: consumes 10 visible → reveals next 10
	//   round 2: consumes 10 visible → reveals next 10
	//   round 3: consumes 5  visible → 25 remaining hidden volume left
	done, _ := ob.Process(matchingo.NewMarketOrder("mkt-buy", matchingo.Buy, matchingo.FromInt(25)))
	fmt.Printf("Market buy 25:\n")
	fmt.Printf("  processed=%s  left=%s  trades=%d\n",
		done.Processed, done.Left, len(done.Trades)-1)
	fmt.Println(ob)
	// Depth now shows 10 units (the next peak) with 15 more hidden.
	fmt.Printf("Depth after: %s\n", ob.DepthJSON())
}
