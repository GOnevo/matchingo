package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.4: Maker entry uses maker price; taker entry uses taker (limit) price.
func TestTradePrice_LimitTakerUsesItsOwnPrice(t *testing.T) {
	ob := newOB()
	// maker SELL at 100; taker BUY limit at 105 — trade price must be 100 (maker)
	ob.Process(matchingo.NewLimitOrder("maker", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("taker", matchingo.Buy, d("5"), d("105"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) < 2 {
		t.Fatalf("expected taker + maker trade entries, got %d", len(done.Trades))
	}
	// trades[0] is taker entry, trades[1] is maker entry
	taker := done.Trades[0]
	maker := done.Trades[1]
	if !taker.Price.Equal(d("105")) {
		t.Fatalf("taker entry price: want 105, got %s", taker.Price)
	}
	if !maker.Price.Equal(d("100")) {
		t.Fatalf("maker entry price: want 100, got %s", maker.Price)
	}
}

// §18.4: MARKET taker trade entry has price zero.
func TestTradePrice_MarketTakerPriceIsZero(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("maker", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewMarketOrder("taker", matchingo.Buy, matchingo.FromInt(5)))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) < 2 {
		t.Fatalf("expected taker + maker trade entries, got %d", len(done.Trades))
	}
	takerEntry := done.Trades[0]
	if !takerEntry.Price.Equal(matchingo.Zero) {
		t.Fatalf("MARKET taker entry price: want 0, got %s", takerEntry.Price)
	}
}
