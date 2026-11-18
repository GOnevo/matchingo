// Example 18 — Self-Trade Prevention (STP)
//
// When an incoming order would trade against a resting order of the same
// participant, the engine applies the incoming order's STP mode instead of
// producing a trade:
//   - STPCancelNewest — cancel the incoming order's remainder
//   - STPCancelOldest — cancel the resting order, keep matching
//   - STPCancelBoth   — cancel the resting order and the incoming remainder
//   - STPDecrement    — reduce both by the overlap, no trade
//
// Concepts shown:
//   - order.SetSTP(participantID, mode)
//   - done.Canceled lists the orders removed by prevention
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	modes := []struct {
		name string
		mode matchingo.STPMode
	}{
		{"CancelNewest", matchingo.STPCancelNewest},
		{"CancelOldest", matchingo.STPCancelOldest},
		{"CancelBoth", matchingo.STPCancelBoth},
		{"Decrement", matchingo.STPDecrement},
	}
	for _, m := range modes {
		ob := matchingo.NewOrderBook()
		own := matchingo.NewLimitOrder("alice-ask", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(10), matchingo.GTC, "")
		own.SetSTP("alice", matchingo.STPNone)
		ob.Process(own)
		ob.Process(matchingo.NewLimitOrder("bob-ask", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(11), matchingo.GTC, ""))

		buy := matchingo.NewLimitOrder("alice-buy", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(11), matchingo.GTC, "")
		buy.SetSTP("alice", m.mode)
		done, _ := ob.Process(buy)
		fmt.Printf("%-13s processed=%s left=%s canceled=%v stored=%v\n", m.name, done.Processed, done.Left, done.Canceled, done.Stored)
	}
}
