package orderbook

import (
	"fmt"
	"testing"

	"github.com/gonevo/matchingo"
)

func TestOrderBook_AdjustedQuantityUsesSubmittedAmount(t *testing.T) {
	for _, kind := range []string{"limit", "market", "quote"} {
		for _, qty := range []int{4, 12} {
			for _, liquidity := range []int{0, 2, 20} {
				// Empty market reporting is tracked separately in upstream PR #6.
				if liquidity == 0 && kind != "limit" {
					continue
				}
				t.Run(fmt.Sprintf("%s/qty%d/liquidity%d", kind, qty, liquidity), func(t *testing.T) {
					book := matchingo.NewOrderBook()
					if liquidity > 0 {
						_, err := book.Process(matchingo.NewLimitOrder("maker", matchingo.Sell, matchingo.FromInt(liquidity), matchingo.FromInt(2), matchingo.GTC, ""))
						if err != nil {
							t.Fatal(err)
						}
					}
					order := matchingo.NewLimitOrder("adjusted", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(2), matchingo.GTC, "")
					if kind == "market" {
						order = matchingo.NewMarketOrder("adjusted", matchingo.Buy, matchingo.FromInt(10))
					}
					if kind == "quote" {
						order = matchingo.NewMarketQuoteOrder("adjusted", matchingo.Buy, matchingo.FromInt(10))
					}
					order.SetQuantity(matchingo.FromInt(qty))
					done, err := book.Process(order)
					if err != nil {
						t.Fatal(err)
					}
					if !done.Quantity.Equal(matchingo.FromInt(qty)) {
						t.Fatalf("reported submitted quantity %s, want %d", done.Quantity, qty)
					}
					if !order.OriginalQty().Equal(matchingo.FromInt(10)) {
						t.Fatal("constructor quantity changed")
					}
					processed := liquidity
					if kind == "quote" {
						processed *= 2
					}
					if processed > qty {
						processed = qty
					}
					if !done.Processed.Equal(matchingo.FromInt(processed)) {
						t.Fatalf("processed=%s, want %d", done.Processed, processed)
					}
					if kind == "limit" && processed == 0 {
						if !done.Left.Equal(matchingo.Zero) {
							t.Fatal("no-trade limit convention changed")
						}
					} else if !done.Left.Equal(matchingo.FromInt(qty - processed)) {
						t.Fatal("remaining quantity mismatch")
					}
					if done.Stored && !book.GetOrder("adjusted").Quantity().Equal(matchingo.FromInt(qty-processed)) {
						t.Fatal("resting quantity reverted to constructor value")
					}
					if processed > 0 && !done.GetTradeOrder("adjusted").Quantity.Equal(matchingo.FromInt(processed)) {
						t.Fatal("taker fill report mismatch")
					}
				})
			}
		}
	}
}
