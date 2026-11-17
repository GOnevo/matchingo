package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.11: Modify unknown ID returns false.
func TestModify_UnknownID(t *testing.T) {
	ob := newOB()
	_, ok := ob.Modify("nonexistent", d("100"), d("5"))
	if ok {
		t.Fatal("Modify unknown ID should return false")
	}
}

// §18.11: Modify a STOP-LIMIT order returns false.
func TestModify_StopLimitReturnsFalse(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("5"), d("105"), d("90"), ""))
	_, ok := ob.Modify("stop1", d("105"), d("3"))
	if ok {
		t.Fatal("Modify stop-limit should return false")
	}
}

// §18.11: In-place decrease keeps time priority — order fills before later orders at same level.
func TestModify_InPlaceDecreasKeepsTimePriority(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("10"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("bid2", matchingo.Buy, d("10"), d("100"), matchingo.GTC, ""))

	// decrease bid1 in place
	_, ok := ob.Modify("bid1", d("100"), d("5"))
	if !ok {
		t.Fatal("in-place decrease should succeed")
	}

	// sell against both; bid1 should fill first (earlier time priority retained)
	done, err := ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if done.GetTradeOrder("bid1") == nil {
		t.Fatal("bid1 should fill first due to retained time priority")
	}
	if done.GetTradeOrder("bid2") != nil {
		t.Fatal("bid2 should not fill yet")
	}
}

// §18.11: In-place decrease updates volume and open amount.
func TestModify_InPlaceDecreaseUpdatesVolume(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("10"), d("100"), matchingo.GTC, ""))

	_, ok := ob.Modify("bid1", d("100"), d("4"))
	if !ok {
		t.Fatal("in-place decrease should succeed")
	}

	o := ob.GetOrder("bid1")
	if !o.Quantity().Equal(d("4")) {
		t.Fatalf("open amount should be 4, got %s", o.Quantity())
	}
	v := ob.Level(matchingo.Buy, d("100"))
	if v == nil || !v.Equal(d("4")) {
		t.Fatalf("volume at 100 should be 4, got %v", v)
	}
}

// §18.11: Price change re-queues, loses time priority.
func TestModify_PriceChangeLosesTimePriority(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("bid2", matchingo.Buy, d("5"), d("101"), matchingo.GTC, ""))

	// change bid2 price to 100 — it should go to back of 100 level
	_, ok := ob.Modify("bid2", d("100"), d("5"))
	if !ok {
		t.Fatal("price change modify should succeed")
	}

	// sell 5 units — should fill bid1 first (earlier at 100)
	done, err := ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if done.GetTradeOrder("bid1") == nil {
		t.Fatal("bid1 should fill first at price 100")
	}
	if done.GetTradeOrder("bid2") != nil {
		t.Fatal("bid2 (re-queued) should not fill yet")
	}
}

// §18.11: Amount increase re-queues, loses time priority.
func TestModify_AmountIncreaseLosesTimePriority(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("bid2", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))

	// increase bid1 amount — it should go to back of 100 level
	_, ok := ob.Modify("bid1", d("100"), d("8"))
	if !ok {
		t.Fatal("amount increase modify should succeed")
	}

	// sell 5 units — should fill bid2 first (bid1 moved to back)
	done, err := ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if done.GetTradeOrder("bid2") == nil {
		t.Fatal("bid2 should fill first; bid1 moved to back after increase")
	}
}

// §18.11: Re-queued order matches immediately when it now crosses.
func TestModify_RequeueMatchesImmediately(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("105"), matchingo.GTC, ""))

	// change ask1 price to 100 — it now crosses bid1 and should match immediately
	done, ok := ob.Modify("ask1", d("100"), d("5"))
	if !ok {
		t.Fatal("modify should succeed")
	}
	if done == nil || len(done.Trades) == 0 {
		t.Fatal("re-queued order should match immediately when it crosses")
	}
}
