package tests

import (
	"testing"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func TestActivatedStopOrdersKeepDistinctIdentities(t *testing.T) {
	for _, side := range []matchingo.Side{matchingo.Buy, matchingo.Sell} {
		t.Run(side.String(), func(t *testing.T) {
			book := matchingo.NewOrderBook()
			firstPrice, secondPrice := int64(9), int64(8)
			opposite := matchingo.Sell
			if side == matchingo.Sell {
				firstPrice, secondPrice = 11, 12
				opposite = matchingo.Buy
			}
			orders := []*matchingo.Order{
				matchingo.NewStopLimitOrder("first", side, fpdecimal.FromInt(2), fpdecimal.FromInt(firstPrice), fpdecimal.FromInt(10), ""),
				matchingo.NewStopLimitOrder("second", side, fpdecimal.FromInt(3), fpdecimal.FromInt(secondPrice), fpdecimal.FromInt(10), ""),
				matchingo.NewLimitOrder("trigger-maker", opposite, fpdecimal.FromInt(1), fpdecimal.FromInt(10), matchingo.GTC, ""),
			}
			for _, order := range orders {
				if _, err := book.Process(order); err != nil {
					t.Fatal(err)
				}
			}
			done, err := book.Process(matchingo.NewMarketOrder("trigger-taker", side, fpdecimal.FromInt(1)))
			if err != nil {
				t.Fatal(err)
			}
			if len(done.Activated) != 2 || done.Activated[0] != "first" || done.Activated[1] != "second" {
				t.Fatalf("activated IDs = %v, want [first second]", done.Activated)
			}
			first, second := book.GetOrder("first"), book.GetOrder("second")
			if first == nil || second == nil || first == second {
				t.Fatal("activated orders must have distinct addresses")
			}
			for i, order := range []*matchingo.Order{first, second} {
				if order.ID() != orders[i].ID() || !order.Quantity().Equal(orders[i].Quantity()) || !order.Price().Equal(orders[i].Price()) || !order.IsLimitOrder() {
					t.Fatalf("activated order %d lost its identity or values: %v", i, order)
				}
			}
			if book.CancelOrder("first") != first || book.GetOrder("second") != second || second.IsCanceled() {
				t.Fatal("canceling the first order affected the second")
			}
			done, err = book.Process(matchingo.NewMarketOrder("fill-second", opposite, fpdecimal.FromInt(3)))
			if err != nil {
				t.Fatal(err)
			}
			trade := done.GetTradeOrder("second")
			if trade == nil || !trade.Quantity.Equal(fpdecimal.FromInt(3)) || !trade.Price.Equal(fpdecimal.FromInt(secondPrice)) || book.GetOrder("second") != nil {
				t.Fatal("the second order could not be filled independently")
			}
		})
	}
}
