// Example 01 — Basic Limit Orders
//
// Demonstrates the simplest use-case: placing resting limit orders on both
// sides of the book and watching them match when prices cross.
//
// Concepts shown:
//   - Creating a new order book
//   - Placing GTC limit orders (Good Till Cancel — the default)
//   - Reading the Done result to see what was traded
//   - Printing the order book state
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// ------------------------------------------------------------------ //
	// Step 1: A seller posts 10 units at $100.                            //
	// No buyer exists yet, so the order rests in the book (Stored=true).  //
	// ------------------------------------------------------------------ //
	sell := matchingo.NewLimitOrder(
		"sell-1",               // unique order ID
		matchingo.Sell,         // side
		matchingo.FromInt(10),  // quantity: 10 units
		matchingo.FromInt(100), // price: $100
		matchingo.GTC,          // time-in-force: Good Till Cancel
		"",                     // OCO pair ID (none)
	)
	done, err := ob.Process(sell)
	fmt.Printf("Sell posted  → stored=%v  err=%v\n", done.Stored, err)
	fmt.Println(ob) // shows asks side with 10 units at 100

	// ------------------------------------------------------------------ //
	// Step 2: A buyer bids 5 units at $100 — price matches immediately.   //
	// The sell order partially fills; 5 units remain resting.             //
	// ------------------------------------------------------------------ //
	buy := matchingo.NewLimitOrder(
		"buy-1",
		matchingo.Buy,
		matchingo.FromInt(5),
		matchingo.FromInt(100),
		matchingo.GTC,
		"",
	)
	done, err = ob.Process(buy)
	fmt.Printf("\nBuy 5 matched → trades=%d  processed=%s  left=%s  err=%v\n",
		len(done.Trades), done.Processed, done.Left, err)
	// Trades[0] is the taker (buy-1); Trades[1] is the maker (sell-1).
	for _, t := range done.Trades {
		fmt.Printf("  trade: order=%s  qty=%s  price=%s  role=%s\n",
			t.OrderID, t.Quantity, t.Price, t.Role)
	}
	fmt.Println(ob) // sell-1 now has only 5 units left

	// ------------------------------------------------------------------ //
	// Step 3: A buyer bids 10 units at $100 but only 5 remain.           //
	// The partial fill empties the ask side; 5 units go unmatched.       //
	// ------------------------------------------------------------------ //
	buy2 := matchingo.NewLimitOrder(
		"buy-2",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(100),
		matchingo.GTC,
		"",
	)
	done, _ = ob.Process(buy2)
	fmt.Printf("\nBuy 10 → processed=%s  left=%s  stored=%v\n",
		done.Processed, done.Left, done.Stored)
	fmt.Println(ob) // empty asks; buy-2 rests with 5 units on the bid side
}
