package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func d(s string) matchingo.Decimal {
	v, _ := matchingo.FromString(s)
	return v
}

func newOB() *matchingo.OrderBook {
	return matchingo.NewOrderBook()
}

func TestSnapshotEmpty(t *testing.T) {
	ob := newOB()
	snap := ob.Dump()
	if snap.Version == 0 {
		t.Fatal("version must be non-zero")
	}
	if len(snap.Bids) != 0 || len(snap.Asks) != 0 || len(snap.Stops) != 0 || len(snap.OCO) != 0 {
		t.Fatal("empty engine dump must have empty lists")
	}
}

func TestSnapshotUnsupportedVersion(t *testing.T) {
	snap := matchingo.Snapshot{Version: 999}
	ob := newOB()
	if err := ob.Restore(snap); err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestSnapshotRestoreClearsState(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("old1", matchingo.Buy, d("1"), d("100"), matchingo.GTC, ""))

	snap := matchingo.Snapshot{Version: 1}
	if err := ob.Restore(snap); err != nil {
		t.Fatal(err)
	}
	if ob.GetOrder("old1") != nil {
		t.Fatal("old orders must be gone after restore")
	}
}

func TestSnapshotLimitOrdersRoundTrip(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("5"), d("99"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, d("3"), d("101"), matchingo.GTC, ""))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	if ob2.GetOrder("b1") == nil {
		t.Fatal("bid order b1 missing after restore")
	}
	if ob2.GetOrder("s1") == nil {
		t.Fatal("ask order s1 missing after restore")
	}

	// matching still works
	done, err := ob2.Process(matchingo.NewLimitOrder("b2", matchingo.Buy, d("3"), d("101"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) == 0 {
		t.Fatal("expected trade after restore")
	}
}

func TestSnapshotStopOrderRoundTrip(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("2"), d("105"), d("103"), ""))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	if ob2.GetOrder("stop1") == nil {
		t.Fatal("stop order missing after restore")
	}
}

func TestSnapshotPartialFillPreserved(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, d("10"), d("100"), matchingo.GTC, ""))
	// partial fill: buy only 3 out of 10
	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("3"), d("100"), matchingo.GTC, ""))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	o := ob2.GetOrder("s1")
	if o == nil {
		t.Fatal("s1 missing after restore")
	}
	if !o.Quantity().Equal(d("7")) {
		t.Fatalf("expected open qty 7, got %s", o.Quantity())
	}
}

func TestSnapshotPostOnlyFlagPreserved(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewPostOnlyOrder("po1", matchingo.Buy, d("5"), d("99"), matchingo.GTC, ""))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	o := ob2.GetOrder("po1")
	if o == nil {
		t.Fatal("post-only order missing after restore")
	}
	if !o.IsPostOnly() {
		t.Fatal("post-only flag not preserved")
	}
}

func TestSnapshotIcebergFieldsPreserved(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Buy, d("10"), d("3"), d("99"), matchingo.GTC, ""))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	o := ob2.GetOrder("ice1")
	if o == nil {
		t.Fatal("iceberg order missing after restore")
	}
	if !o.IsIceberg() {
		t.Fatal("iceberg flag not preserved")
	}
	if !o.PeakQty().Equal(d("3")) {
		t.Fatalf("expected peak 3, got %s", o.PeakQty())
	}
	if !o.HiddenQty().Equal(d("7")) {
		t.Fatalf("expected hidden 7, got %s", o.HiddenQty())
	}
}

func TestSnapshotOCOLinkPreserved(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("a1", matchingo.Buy, d("5"), d("99"), matchingo.GTC, "a2"))
	ob.Process(matchingo.NewLimitOrder("a2", matchingo.Sell, d("5"), d("101"), matchingo.GTC, "a1"))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	// filling a1 should cancel a2
	done, err := ob2.Process(matchingo.NewLimitOrder("taker", matchingo.Sell, d("5"), d("99"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	canceled := false
	for _, c := range done.Canceled {
		if c == "a2" {
			canceled = true
		}
	}
	if !canceled {
		t.Fatal("OCO cancel did not propagate after restore")
	}
}
