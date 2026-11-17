// Example 05 — Stop-Limit Orders
//
// A stop-limit order is inactive until the market price crosses the stop
// price, at which point it becomes a regular limit order and is matched.
//
// Stop direction is determined by order side:
//   - Sell stop → activates when market trades AT OR BELOW the stop price
//   - Buy  stop → activates when market trades AT OR ABOVE the stop price
//
// Workflow:
//  1. Place a stop-limit order  → it rests in the stop book (not visible)
//  2. A trade occurs            → engine checks stop book for triggered orders
//  3. Triggered orders activate → they enter the live book and may match
//
// Concepts shown:
//   - matchingo.NewStopLimitOrder
//   - Done.Activated lists order IDs that became live after a trade
//   - Done.Activations carries the Done result of each activated order
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// ------------------------------------------------------------------ //
	// Place a stop-loss sell: "if price drops to $95, sell at $94".       //
	// The order sits in the stop book — invisible to the depth feed.       //
	// ------------------------------------------------------------------ //
	ob.Process(matchingo.NewStopLimitOrder(
		"stop-sell",
		matchingo.Sell,
		matchingo.FromInt(5),  // quantity
		matchingo.FromInt(94), // limit price (minimum acceptable sell price)
		matchingo.FromInt(95), // stop price  (activation threshold)
		"",
	))
	fmt.Println("Book after placing stop-sell (stop book only, not visible in depth):")
	fmt.Println(ob) // empty live book

	// ------------------------------------------------------------------ //
	// Seed liquidity: a buyer willing to buy at $94.                      //
	// This will be the counterparty once the stop activates.              //
	// ------------------------------------------------------------------ //
	ob.Process(matchingo.NewLimitOrder("bid-ready", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(94), matchingo.GTC, ""))
	fmt.Println("Bid at $94 added (will absorb stop-sell once it activates):")
	fmt.Println(ob)

	// ------------------------------------------------------------------ //
	// Trigger: a trade at $95 crosses the stop price.                     //
	// Step 1 — the trigger order itself matches existing liquidity.       //
	// Step 2 — the engine scans the stop book and activates stop-sell.    //
	// Step 3 — the now-live stop-sell matches the resting bid-ready.      //
	// ------------------------------------------------------------------ //
	ob.Process(matchingo.NewLimitOrder("ask-trig", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(95), matchingo.GTC, ""))
	done, _ := ob.Process(matchingo.NewLimitOrder("buy-trig", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(95), matchingo.GTC, ""))

	fmt.Printf("\nTrigger trade at $95:\n")
	fmt.Printf("  trig processed=%s  activated=%v\n", done.Processed, done.Activated)
	for _, act := range done.Activations {
		fmt.Printf("  activation %s → processed=%s  left=%s  stored=%v\n",
			act.Order.ID(), act.Processed, act.Left, act.Stored)
	}
	fmt.Println(ob) // stop-sell matched bid-ready; both should be gone

	// ------------------------------------------------------------------ //
	// Breakout buy example: "if price rises to $110, buy at $111".        //
	// ------------------------------------------------------------------ //
	ob2 := matchingo.NewOrderBook()
	ob2.Process(matchingo.NewStopLimitOrder("stop-buy", matchingo.Buy, matchingo.FromInt(4), matchingo.FromInt(111), matchingo.FromInt(110), ""))
	// Seed a sell at $111 so the activated stop-buy has liquidity to consume.
	ob2.Process(matchingo.NewLimitOrder("ask-ready", matchingo.Sell, matchingo.FromInt(4), matchingo.FromInt(111), matchingo.GTC, ""))

	// Seed a bid at $110 so a crossing sell can trigger the trade.
	ob2.Process(matchingo.NewLimitOrder("bid-trig", matchingo.Buy, matchingo.FromInt(2), matchingo.FromInt(110), matchingo.GTC, ""))
	// A sell at $110 crosses bid-trig → trade at $110 → activates stop-buy.
	done2, _ := ob2.Process(matchingo.NewLimitOrder("ask-trig2", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(110), matchingo.GTC, ""))

	fmt.Printf("\nBreakout at $110 → activated=%v\n", done2.Activated)
	for _, act := range done2.Activations {
		fmt.Printf("  activation %s → processed=%s\n", act.Order.ID(), act.Processed)
	}
}
