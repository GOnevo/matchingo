package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.10: Post-only rests when book is empty.
func TestPostOnly_RestsWhenBookEmpty(t *testing.T) {
	ob := newOB()
	done, err := ob.Process(matchingo.NewPostOnlyOrder("po1", matchingo.Buy, d("5"), d("99"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Canceled) != 0 {
		t.Fatal("post-only should not be canceled when book is empty")
	}
	if ob.GetOrder("po1") == nil {
		t.Fatal("post-only should rest in book")
	}
}

// §18.10: Post-only rests when buy price is below best ask.
func TestPostOnly_RestsWhenBelowBestAsk(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("105"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewPostOnlyOrder("po1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Canceled) != 0 {
		t.Fatal("post-only should not be canceled when below best ask")
	}
	if ob.GetOrder("po1") == nil {
		t.Fatal("post-only should rest in book")
	}
}

// §18.10: Post-only canceled when buy price crosses ask.
func TestPostOnly_CanceledWhenBuyCrossesAsk(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewPostOnlyOrder("po1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Canceled) == 0 {
		t.Fatal("post-only should be canceled when it would cross ask")
	}
	if len(done.Trades) != 0 {
		t.Fatal("post-only should not trade when canceled")
	}
	// resting ask should be untouched
	if ob.GetOrder("ask1") == nil {
		t.Fatal("resting ask should be untouched")
	}
}

// §18.10: Post-only canceled when sell price crosses bid.
func TestPostOnly_CanceledWhenSellCrossesBid(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewPostOnlyOrder("po1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Canceled) == 0 {
		t.Fatal("post-only should be canceled when it would cross bid")
	}
	if len(done.Trades) != 0 {
		t.Fatal("post-only should not trade when canceled")
	}
	if ob.GetOrder("bid1") == nil {
		t.Fatal("resting bid should be untouched")
	}
}

// §18.10/§18.13: Post-only survives snapshot round-trip.
func TestPostOnly_SnapshotRoundTrip(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewPostOnlyOrder("po1", matchingo.Buy, d("5"), d("99"), matchingo.GTC, ""))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	o := ob2.GetOrder("po1")
	if o == nil {
		t.Fatal("post-only missing after restore")
	}
	if !o.IsPostOnly() {
		t.Fatal("post-only flag not preserved after restore")
	}

	// add a fresh ask above the restored bid so it doesn't match po1
	ob2.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))
	// po2 BUY at 100 would cross ask1 — post-only must be canceled
	done, err := ob2.Process(matchingo.NewPostOnlyOrder("po2", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Canceled) == 0 {
		t.Fatal("post-only behavior should be preserved after restore")
	}
}
