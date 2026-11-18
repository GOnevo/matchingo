// Example 19 — Atomic Replace (cancel-replace)
//
// Replace swaps a resting order for a new one in a single call. The new order
// is validated first, so a bad replacement leaves the old order untouched;
// otherwise the old order is canceled and the new one processed with nothing in
// between. The new order may reuse the old ID and always queues at the back.
//
// Concepts shown:
//   - ob.Replace(oldID, newOrder)
//   - done.Canceled lists the replaced order first
//   - A failed Replace leaves the book unchanged
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook(matchingo.WithPrecision(2, 2))
	ob.Process(matchingo.NewLimitOrder("bid", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), matchingo.GTC, ""))

	done, err := ob.Replace("bid", matchingo.NewLimitOrder("bid", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(101), matchingo.GTC, ""))
	fmt.Printf("replace: err=%v canceled=%v stored=%v\n", err, done.Canceled, done.Stored)

	_, err = ob.Replace("bid", matchingo.NewLimitOrder("bid", matchingo.Buy, matchingo.FromFloat(1.005), matchingo.FromInt(101), matchingo.GTC, ""))
	fmt.Printf("invalid replace: err=%v, old order still resting: %v\n", err != nil, ob.GetOrder("bid") != nil)
}
