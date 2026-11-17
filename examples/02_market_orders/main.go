// Example 02 — Market Orders
//
// Market orders execute immediately at the best available price.
// They have no price field; matching continues until the quantity is filled
// or the opposite side is exhausted.
//
// Two market order variants:
//   - Regular market order  — quantity in base currency (e.g. "buy 5 BTC")
//   - Quote  market order   — quantity in quote currency (e.g. "spend $500")
//
// Concepts shown:
//   - matchingo.NewMarketOrder
//   - matchingo.NewMarketQuoteOrder
//   - How Done.Left signals an unfilled remainder
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Seed the book with a few sell levels so the market order has liquidity.
	ob.Process(matchingo.NewLimitOrder("ask-1", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(100), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-2", matchingo.Sell, matchingo.FromInt(4), matchingo.FromInt(102), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-3", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(105), matchingo.GTC, ""))
	fmt.Println("Book seeded:")
	fmt.Println(ob)

	// ------------------------------------------------------------------ //
	// Market buy: consume 6 units across two price levels.                //
	// 3 units from ask-1 at $100, then 3 units from ask-2 at $102.       //
	// ------------------------------------------------------------------ //
	done, _ := ob.Process(matchingo.NewMarketOrder("mkt-buy-1", matchingo.Buy, matchingo.FromInt(6)))
	fmt.Printf("Market buy 6 → processed=%s  left=%s  trades=%d\n",
		done.Processed, done.Left, len(done.Trades)-1) // -1 for the taker entry
	fmt.Println(ob) // ask-1 gone; ask-2 has 1 unit left

	// ------------------------------------------------------------------ //
	// Market buy with more qty than available liquidity.                  //
	// The order fills what it can; Left > 0 shows what went unmatched.   //
	// ------------------------------------------------------------------ //
	done, _ = ob.Process(matchingo.NewMarketOrder("mkt-buy-2", matchingo.Buy, matchingo.FromInt(100)))
	fmt.Printf("\nMarket buy 100 → processed=%s  left=%s\n", done.Processed, done.Left)
	fmt.Println(ob) // book is now empty

	// ------------------------------------------------------------------ //
	// Seed asks again for the quote-mode example.                         //
	// ------------------------------------------------------------------ //
	ob.Process(matchingo.NewLimitOrder("ask-4", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(50), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-5", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(60), matchingo.GTC, ""))

	// ------------------------------------------------------------------ //
	// Quote market order: "spend up to $400 worth of this asset".         //
	// At $50/unit we expect to buy 8 units (8×50=400).                   //
	// ------------------------------------------------------------------ //
	done, _ = ob.Process(matchingo.NewMarketQuoteOrder("mkt-quote-1", matchingo.Buy, matchingo.FromInt(400)))
	fmt.Printf("\nMarket quote buy $400 → processed=%s  left=%s  trades=%d\n",
		done.Processed, done.Left, len(done.Trades)-1)
	fmt.Println(ob) // ask-4 has 2 units left; ask-5 untouched
}
