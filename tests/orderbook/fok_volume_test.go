package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.15: FOK BUY — sum of SELL volume at or below limit price is enough → fills.
func TestFOKVolume_BuyPassesWhenEnoughCrossingVolume(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask2", matchingo.Sell, d("5"), d("102"), matchingo.GTC, ""))

	// FOK BUY limit 102, wants 10 → 5+5=10 available at ≤102 → passes
	done, err := ob.Process(matchingo.NewLimitOrder("fok1", matchingo.Buy, d("10"), d("102"), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) == 0 {
		t.Fatal("FOK BUY should fill when enough crossing volume")
	}
}

// §18.15: FOK SELL — sum of BUY volume at or above limit price is enough → fills.
func TestFOKVolume_SellPassesWhenEnoughCrossingVolume(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("bid2", matchingo.Buy, d("5"), d("98"), matchingo.GTC, ""))

	// FOK SELL limit 98, wants 10 → 5+5=10 available at ≥98 → passes
	done, err := ob.Process(matchingo.NewLimitOrder("fok1", matchingo.Sell, d("10"), d("98"), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) == 0 {
		t.Fatal("FOK SELL should fill when enough crossing volume")
	}
}

// §18.15: FOK BUY — SELL volume at out-of-range prices is excluded.
func TestFOKVolume_BuyExcludesOutOfRangeSells(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask-in", matchingo.Sell, d("3"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-out", matchingo.Sell, d("10"), d("105"), matchingo.GTC, ""))

	// FOK BUY limit 102, wants 10 → only 3 units at ≤102 → fails
	done, err := ob.Process(matchingo.NewLimitOrder("fok1", matchingo.Buy, d("10"), d("102"), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) != 0 {
		t.Fatal("FOK BUY should be canceled when out-of-range volume is excluded")
	}
	if len(done.Canceled) == 0 {
		t.Fatal("FOK BUY should have cancel entry when it fails")
	}
}

// §18.15: FOK SELL — BUY volume at out-of-range prices is excluded.
func TestFOKVolume_SellExcludesOutOfRangeBuys(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid-in", matchingo.Buy, d("3"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("bid-out", matchingo.Buy, d("10"), d("95"), matchingo.GTC, ""))

	// FOK SELL limit 98, wants 10 → only 3 units at ≥98 → fails
	done, err := ob.Process(matchingo.NewLimitOrder("fok1", matchingo.Sell, d("10"), d("98"), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) != 0 {
		t.Fatal("FOK SELL should be canceled when out-of-range volume is excluded")
	}
	if len(done.Canceled) == 0 {
		t.Fatal("FOK SELL should have cancel entry when it fails")
	}
}
