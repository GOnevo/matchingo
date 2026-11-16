package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestOrderBook_OCOCancelsSiblingExactlyOnce(t *testing.T) {
	for _, side := range []matchingo.Side{matchingo.Buy, matchingo.Sell} {
		for _, kind := range []string{"limit", "pending-stop", "activated-stop"} {
			t.Run(side.String()+"/"+kind, func(t *testing.T) {
				book := matchingo.NewOrderBook()
				siblingPrice, triggerPrice, untouchedPrice := int64(8), int64(20), int64(7)
				opposite := matchingo.Sell
				if side == matchingo.Sell {
					siblingPrice, triggerPrice, untouchedPrice = 12, 5, 13
					opposite = matchingo.Buy
				}
				process := func(order *matchingo.Order) *matchingo.Done {
					t.Helper()
					done, err := book.Process(order)
					if err != nil {
						t.Fatal(err)
					}
					return done
				}
				process(matchingo.NewLimitOrder("primary", side, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, "sibling"))
				process(matchingo.NewLimitOrder("untouched", side, matchingo.FromInt(5), matchingo.FromInt(int(untouchedPrice)), matchingo.GTC, ""))
				if kind == "limit" {
					process(matchingo.NewLimitOrder("sibling", side, matchingo.FromInt(2), matchingo.FromInt(int(siblingPrice)), matchingo.GTC, "primary"))
				} else {
					process(matchingo.NewStopLimitOrder("sibling", side, matchingo.FromInt(2), matchingo.FromInt(int(siblingPrice)), matchingo.FromInt(int(triggerPrice)), "primary"))
					if kind == "activated-stop" {
						process(matchingo.NewLimitOrder("trigger-maker", opposite, matchingo.FromInt(1), matchingo.FromInt(int(triggerPrice)), matchingo.GTC, ""))
						process(matchingo.NewMarketOrder("trigger-taker", side, matchingo.FromInt(1)))
						if !book.GetOrder("sibling").IsLimitOrder() {
							t.Fatal("stop sibling did not activate")
						}
					}
				}
				sibling := book.GetOrder("sibling")
				done := process(matchingo.NewMarketOrder("fill-primary", opposite, matchingo.FromInt(1)))
				if len(done.Canceled) != 1 || done.Canceled[0] != "sibling" {
					t.Fatalf("canceled IDs = %v, want [sibling]", done.Canceled)
				}
				if book.GetOrder("primary") != nil || book.GetOrder("sibling") != nil || !sibling.IsCanceled() {
					t.Fatal("filled primary or canceled sibling is still live")
				}
				if book.Stop.Len() != 0 || len(book.OCO) != 0 {
					t.Fatalf("unexpected remaining state: stops=%d, OCO=%v", book.Stop.Len(), book.OCO)
				}
				trade := done.GetTradeOrder("primary")
				if trade == nil || !trade.Quantity.Equal(matchingo.FromInt(1)) || done.GetTradeOrder("sibling") != nil {
					t.Fatal("unexpected trade accounting")
				}
				price, err := book.CalculateMarketPrice(opposite, matchingo.FromInt(5))
				if err != nil || !price.Equal(matchingo.FromInt(int(untouchedPrice*5))) {
					t.Fatalf("unrelated liquidity changed: price=%s, err=%v", price, err)
				}
			})
		}
	}
}
