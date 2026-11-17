// Example 04 — Cancel and Modify
//
// Orders resting in the book can be canceled entirely or modified in place.
//
// Modify rules:
//   - Price change OR quantity increase → re-queued at the back of the new
//     price level, then matched (loses time priority).
//   - Quantity decrease (same price)   → reduced in place, keeps time priority.
//
// Concepts shown:
//   - ob.Cancel(id)
//   - ob.Modify(id, newPrice, newQty)
//   - Time-priority implications of re-queuing
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Place two resting sell orders.
	ob.Process(matchingo.NewLimitOrder("sell-A", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("sell-B", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), matchingo.GTC, ""))
	fmt.Println("Initial state:")
	fmt.Println(ob) // two 10-unit asks at $100

	// ------------------------------------------------------------------ //
	// Cancel sell-A.                                                      //
	// ------------------------------------------------------------------ //
	ok := ob.Cancel("sell-A")
	fmt.Printf("Cancel sell-A → found=%v\n", ok)
	fmt.Println(ob) // only sell-B remains

	// ------------------------------------------------------------------ //
	// In-place quantity decrease on sell-B (same price, smaller qty).    //
	// Keeps time priority — no re-queue.                                  //
	// ------------------------------------------------------------------ //
	done, ok := ob.Modify("sell-B", matchingo.FromInt(100), matchingo.FromInt(6))
	fmt.Printf("\nModify sell-B qty → 6  (in-place)  done=%v  matched=%v\n", ok, done != nil)
	fmt.Println(ob) // sell-B now shows 6 units at $100

	// ------------------------------------------------------------------ //
	// Price change on sell-B → re-queued at new price, then matched.     //
	// A buyer at $98 is waiting; the new price of $98 crosses it.         //
	// ------------------------------------------------------------------ //
	ob.Process(matchingo.NewLimitOrder("bid-1", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(98), matchingo.GTC, ""))
	fmt.Println("\nBid at $98 resting:")
	fmt.Println(ob)

	done, ok = ob.Modify("sell-B", matchingo.FromInt(98), matchingo.FromInt(6))
	fmt.Printf("\nModify sell-B price → $98  (re-queue + match)  ok=%v\n", ok)
	if done != nil {
		fmt.Printf("  trades=%d  processed=%s  left=%s  stored=%v\n",
			len(done.Trades)-1, done.Processed, done.Left, done.Stored)
	}
	fmt.Println(ob) // sell-B has 3 units left at $98; bid-1 is gone
}
