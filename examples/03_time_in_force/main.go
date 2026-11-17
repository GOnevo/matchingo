// Example 03 — Time-in-Force (GTC / IOC / FOK)
//
// Time-in-Force controls what happens to the unfilled portion of an order.
//
//	GTC (Good Till Cancel) — rests in the book until manually canceled.
//	IOC (Immediate or Cancel) — fills what it can right now; cancels the rest.
//	FOK (Fill or Kill)       — must fill completely or the whole order is rejected.
//
// Concepts shown:
//   - matchingo.GTC, matchingo.IOC, matchingo.FOK constants
//   - Done.Stored signals whether the order rests in the book
//   - Done.Left > 0 with IOC means a partial fill; with FOK it means no fill
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Seed: 5 units available at $100.
	ob.Process(matchingo.NewLimitOrder("ask-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	fmt.Println("Book seeded with 5 units at $100:")
	fmt.Println(ob)

	// ------------------------------------------------------------------ //
	// IOC buy for 8 units — only 5 are available.                        //
	// 5 units fill; the remaining 3 are immediately canceled (not stored).//
	// ------------------------------------------------------------------ //
	done, _ := ob.Process(matchingo.NewLimitOrder(
		"ioc-1", matchingo.Buy, matchingo.FromInt(8), matchingo.FromInt(100), matchingo.IOC, "",
	))
	fmt.Printf("IOC buy 8 → processed=%s  left=%s  stored=%v\n",
		done.Processed, done.Left, done.Stored)
	// Left=3 (unfilled), Stored=false (not placed on the book)
	fmt.Println(ob) // book should be empty now

	// Refill: 3 units at $100.
	ob.Process(matchingo.NewLimitOrder("ask-2", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(100), matchingo.GTC, ""))

	// ------------------------------------------------------------------ //
	// FOK buy for 5 units — only 3 are available.                        //
	// The entire order is rejected (zero trades, nothing stored).         //
	// ------------------------------------------------------------------ //
	done, _ = ob.Process(matchingo.NewLimitOrder(
		"fok-1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.FOK, "",
	))
	fmt.Printf("\nFOK buy 5 → processed=%s  left=%s  stored=%v  trades=%d\n",
		done.Processed, done.Left, done.Stored, len(done.Trades))
	// processed=0, left=5, stored=false, trades=0
	fmt.Println(ob) // ask-2 still intact — FOK did not consume anything

	// ------------------------------------------------------------------ //
	// FOK buy for 3 units — exactly meets available liquidity.            //
	// This one fills completely.                                           //
	// ------------------------------------------------------------------ //
	done, _ = ob.Process(matchingo.NewLimitOrder(
		"fok-2", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(100), matchingo.FOK, "",
	))
	fmt.Printf("\nFOK buy 3 → processed=%s  left=%s  stored=%v  trades=%d\n",
		done.Processed, done.Left, done.Stored, len(done.Trades))
	fmt.Println(ob) // empty — all 3 units matched
}
