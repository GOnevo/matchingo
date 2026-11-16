package tests

import (
	"fmt"
	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
	"testing"
)

func TestAdjustedQuantityUsesSubmittedAmount(t *testing.T) {
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
						_, err := book.Process(matchingo.NewLimitOrder("maker", matchingo.Sell, fpdecimal.FromInt(liquidity), fpdecimal.FromInt(2), matchingo.GTC, ""))
						if err != nil {
							t.Fatal(err)
						}
					}
					order := matchingo.NewLimitOrder("adjusted", matchingo.Buy, fpdecimal.FromInt(10), fpdecimal.FromInt(2), matchingo.GTC, "")
					if kind == "market" {
						order = matchingo.NewMarketOrder("adjusted", matchingo.Buy, fpdecimal.FromInt(10))
					}
					if kind == "quote" {
						order = matchingo.NewMarketQuoteOrder("adjusted", matchingo.Buy, fpdecimal.FromInt(10))
					}
					order.SetQuantity(fpdecimal.FromInt(qty))
					done, err := book.Process(order)
					if err != nil {
						t.Fatal(err)
					}
					if !done.Quantity.Equal(fpdecimal.FromInt(qty)) {
						t.Fatalf("reported submitted quantity %s, want %d", done.Quantity, qty)
					}
					if !order.OriginalQty().Equal(fpdecimal.FromInt(10)) {
						t.Fatal("constructor quantity changed")
					}
					processed := liquidity
					if kind == "quote" {
						processed *= 2
					}
					if processed > qty {
						processed = qty
					}
					if !done.Processed.Equal(fpdecimal.FromInt(processed)) {
						t.Fatalf("processed=%s, want %d", done.Processed, processed)
					}
					if kind == "limit" && processed == 0 {
						if !done.Left.Equal(fpdecimal.Zero) {
							t.Fatal("no-trade limit convention changed")
						}
					} else if !done.Left.Equal(fpdecimal.FromInt(qty - processed)) {
						t.Fatal("remaining quantity mismatch")
					}
					if done.Stored && !book.GetOrder("adjusted").Quantity().Equal(fpdecimal.FromInt(qty-processed)) {
						t.Fatal("resting quantity reverted to constructor value")
					}
					if processed > 0 && !done.GetTradeOrder("adjusted").Quantity.Equal(fpdecimal.FromInt(processed)) {
						t.Fatal("taker fill report mismatch")
					}
				})
			}
		}
	}
}
