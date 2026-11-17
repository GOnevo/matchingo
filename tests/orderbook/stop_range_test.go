package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.7: BUY stop activates when trade price jumps over stop price (range, not exact).
func TestStop_BUYActivatesWhenPriceJumpsOverStop(t *testing.T) {
	ob := newOB()
	// BUY stop at stop price 95; trigger a trade at 100 (above 95)
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("2"), d("110"), d("95"), ""))
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("1"), d("100"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("1"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Activated) == 0 {
		t.Fatal("BUY stop should activate when trade price (100) >= stop price (95)")
	}
}

// §18.7: SELL stop activates when trade price jumps below stop price (range, not exact).
func TestStop_SELLActivatesWhenPriceJumpsUnderStop(t *testing.T) {
	ob := newOB()
	// SELL stop at stop price 105; trigger a trade at 100 (below 105)
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Sell, d("2"), d("90"), d("105"), ""))
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("1"), d("100"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("1"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Activated) == 0 {
		t.Fatal("SELL stop should activate when trade price (100) <= stop price (105)")
	}
}

// §18.7: Stop does not fire when no trade happens (resting order placed, no match).
func TestStop_DoesNotFireWithoutTrade(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("2"), d("110"), d("90"), ""))
	// place a resting order that doesn't match anything
	ob.Process(matchingo.NewLimitOrder("rest1", matchingo.Buy, d("2"), d("80"), matchingo.GTC, ""))
	if ob.GetOrder("stop1") == nil {
		t.Fatal("stop1 should still be in stop book after a non-trade event")
	}
}

// §18.7: Activated stop can trigger more stops (cascade).
func TestStop_Cascade(t *testing.T) {
	ob := newOB()
	// stop1 activates at 90, its limit is 110 → will fill ask2 and cause a trade at 100
	// stop2 activates when price >= 95 (between 90 and 100)
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("1"), d("110"), d("90"), ""))
	ob.Process(matchingo.NewStopLimitOrder("stop2", matchingo.Buy, d("1"), d("120"), d("95"), ""))
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("1"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask2", matchingo.Sell, d("1"), d("100"), matchingo.GTC, ""))

	// initial trigger at 90
	ob.Process(matchingo.NewLimitOrder("askTrig", matchingo.Sell, d("1"), d("90"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("bidTrig", matchingo.Buy, d("1"), d("90"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}

	// stop1 activates from trigger. stop1 fills at 100, which may cascade stop2.
	// At minimum stop1 must be in activated list.
	if len(done.Activated) == 0 {
		t.Fatal("stop1 should be activated by initial trigger")
	}

	// stop2 should cascade (activated from stop1's trade)
	activated := map[string]bool{}
	activated[done.Activated[0]] = true
	for _, act := range done.Activations {
		for _, id := range act.Activated {
			activated[id] = true
		}
	}
	if !activated["stop1"] {
		t.Fatal("stop1 should be in activated set")
	}
	if !activated["stop2"] {
		t.Fatal("stop2 should cascade-activate from stop1's trade")
	}
}

// §18.7: BUY stops activate lowest stop price first.
func TestStop_BUYActivationOrderLowestFirst(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewStopLimitOrder("high-stop", matchingo.Buy, d("1"), d("200"), d("100"), ""))
	ob.Process(matchingo.NewStopLimitOrder("low-stop", matchingo.Buy, d("1"), d("200"), d("90"), ""))

	ob.Process(matchingo.NewLimitOrder("askTrig", matchingo.Sell, d("1"), d("100"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("bidTrig", matchingo.Buy, d("1"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Activated) < 2 {
		t.Fatalf("both stops should activate, got %v", done.Activated)
	}
	if done.Activated[0] != "low-stop" {
		t.Fatalf("lowest stop price should activate first, got %v", done.Activated)
	}
	if done.Activated[1] != "high-stop" {
		t.Fatalf("highest stop price should activate second, got %v", done.Activated)
	}
}

// §18.7: SELL stops activate highest stop price first.
func TestStop_SELLActivationOrderHighestFirst(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewStopLimitOrder("low-stop", matchingo.Sell, d("1"), d("1"), d("90"), ""))
	ob.Process(matchingo.NewStopLimitOrder("high-stop", matchingo.Sell, d("1"), d("1"), d("100"), ""))

	ob.Process(matchingo.NewLimitOrder("bidTrig", matchingo.Buy, d("1"), d("90"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("askTrig", matchingo.Sell, d("1"), d("90"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Activated) < 2 {
		t.Fatalf("both stops should activate, got %v", done.Activated)
	}
	if done.Activated[0] != "high-stop" {
		t.Fatalf("highest stop price should activate first, got %v", done.Activated)
	}
	if done.Activated[1] != "low-stop" {
		t.Fatalf("lowest stop price should activate second, got %v", done.Activated)
	}
}

// §18.7: Within same stop price, insertion order is preserved.
func TestStop_SameStopPriceInsertionOrder(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewStopLimitOrder("first", matchingo.Buy, d("1"), d("200"), d("90"), ""))
	ob.Process(matchingo.NewStopLimitOrder("second", matchingo.Buy, d("1"), d("200"), d("90"), ""))

	ob.Process(matchingo.NewLimitOrder("askTrig", matchingo.Sell, d("1"), d("90"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("bidTrig", matchingo.Buy, d("1"), d("90"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Activated) < 2 {
		t.Fatalf("both stops should activate, got %v", done.Activated)
	}
	if done.Activated[0] != "first" || done.Activated[1] != "second" {
		t.Fatalf("insertion order not preserved: %v", done.Activated)
	}
}
