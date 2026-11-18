package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestMarketToLimit_PartialFillRestsAtLastTradePrice(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewLimitOrder("a1", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))
	b.Process(matchingo.NewLimitOrder("a2", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(12), matchingo.GTC, ""))

	done, err := b.Process(matchingo.NewMarketToLimitOrder("mtl", matchingo.Buy, matchingo.FromInt(5)))
	if err != nil {
		t.Fatal(err)
	}
	if !done.Processed.Equal(matchingo.FromInt(2)) || !done.Left.Equal(matchingo.FromInt(3)) || !done.Stored || len(done.Canceled) != 0 {
		t.Fatalf("processed=%s left=%s stored=%v canceled=%v", done.Processed, done.Left, done.Stored, done.Canceled)
	}
	o := b.GetOrder("mtl")
	if o == nil || !o.IsLimitOrder() || !o.Price().Equal(matchingo.FromInt(12)) || !o.Quantity().Equal(matchingo.FromInt(3)) {
		t.Fatalf("remainder wrong: %v", o)
	}
	if bid := b.BestBid(); bid == nil || !bid.Price.Equal(matchingo.FromInt(12)) {
		t.Fatal("remainder should rest as best bid at 12")
	}
	// it behaves as a normal resting limit order afterwards
	d, _ := b.Process(matchingo.NewMarketOrder("sell", matchingo.Sell, matchingo.FromInt(3)))
	if tr := d.GetTradeOrder("mtl"); tr == nil || !tr.Price.Equal(matchingo.FromInt(12)) {
		t.Fatal("remainder not fillable at 12")
	}
}

func TestMarketToLimit_FullFillAndEmptyBook(t *testing.T) {
	b := matchingo.NewOrderBook()
	done, _ := b.Process(matchingo.NewMarketToLimitOrder("none", matchingo.Buy, matchingo.FromInt(1)))
	if done.Stored || len(done.Canceled) != 1 || b.GetOrder("none") != nil {
		t.Fatal("no liquidity: order must be canceled")
	}
	b.Process(matchingo.NewLimitOrder("a", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(10), matchingo.GTC, ""))
	done, _ = b.Process(matchingo.NewMarketToLimitOrder("full", matchingo.Buy, matchingo.FromInt(2)))
	if done.Stored || !done.Left.Equal(Zero) || b.GetOrder("full") != nil {
		t.Fatal("fully filled order must not rest")
	}
}
