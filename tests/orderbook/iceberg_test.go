package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.9: Peak amount ≥ total amount is rejected.
func TestIceberg_InvalidPeakRejected(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when peak >= total")
		}
	}()
	matchingo.NewIcebergOrder("ice1", matchingo.Buy, d("5"), d("5"), d("100"), matchingo.GTC, "")
}

// §18.9: Open amount starts at peak amount.
func TestIceberg_OpenAmountStartsAtPeak(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Buy, d("10"), d("3"), d("100"), matchingo.GTC, ""))

	o := ob.GetOrder("ice1")
	if o == nil {
		t.Fatal("iceberg order not in book")
	}
	if !o.Quantity().Equal(d("3")) {
		t.Fatalf("expected visible qty 3, got %s", o.Quantity())
	}
}

// §18.9: Partial fill within peak — peak reduced, order stays.
func TestIceberg_PartialFillWithinPeak(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Sell, d("10"), d("3"), d("100"), matchingo.GTC, ""))

	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("2"), d("100"), matchingo.GTC, ""))

	o := ob.GetOrder("ice1")
	if o == nil {
		t.Fatal("iceberg should still be in book")
	}
	if !o.Quantity().Equal(d("1")) {
		t.Fatalf("expected remaining peak qty 1, got %s", o.Quantity())
	}
	if !o.HiddenQty().Equal(d("7")) {
		t.Fatalf("expected hidden qty 7, got %s", o.HiddenQty())
	}
}

// §18.9: Full peak exhausted — refills to next peak.
func TestIceberg_FullPeakExhaustedRefills(t *testing.T) {
	ob := newOB()
	// total=10, peak=3 → first peak 3, hidden 7
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Sell, d("10"), d("3"), d("100"), matchingo.GTC, ""))

	// exhaust first peak exactly
	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("3"), d("100"), matchingo.GTC, ""))

	o := ob.GetOrder("ice1")
	if o == nil {
		t.Fatal("iceberg should be re-inserted after refill")
	}
	if !o.Quantity().Equal(d("3")) {
		t.Fatalf("expected refilled peak qty 3, got %s", o.Quantity())
	}
	if !o.HiddenQty().Equal(d("4")) {
		t.Fatalf("expected hidden qty 4 after refill, got %s", o.HiddenQty())
	}
}

// §18.9: Last peak is smaller than peak size.
func TestIceberg_LastPeakSmallerThanPeakSize(t *testing.T) {
	ob := newOB()
	// total=7, peak=3 → peaks: 3, 3, 1
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Sell, d("7"), d("3"), d("100"), matchingo.GTC, ""))

	// exhaust first two peaks
	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("3"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("b2", matchingo.Buy, d("3"), d("100"), matchingo.GTC, ""))

	o := ob.GetOrder("ice1")
	if o == nil {
		t.Fatal("iceberg should still be in book with last peak")
	}
	if !o.Quantity().Equal(d("1")) {
		t.Fatalf("expected last peak qty 1, got %s", o.Quantity())
	}
	if !o.HiddenQty().Equal(d("0")) {
		t.Fatalf("expected hidden qty 0, got %s", o.HiddenQty())
	}
}

// §18.9: Fully exhausted — removed from book.
func TestIceberg_FullyExhaustedRemoved(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Sell, d("6"), d("3"), d("100"), matchingo.GTC, ""))

	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("6"), d("100"), matchingo.GTC, ""))

	if ob.GetOrder("ice1") != nil {
		t.Fatal("iceberg should be gone after full exhaustion")
	}
}

// §18.9: Taker sweeps iceberg across multiple refills.
func TestIceberg_TakerSweepsMultipleRefills(t *testing.T) {
	ob := newOB()
	// total=9, peak=3 → 3 peaks
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Sell, d("9"), d("3"), d("100"), matchingo.GTC, ""))

	done, err := ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("9"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) < 4 { // taker entry + 3 maker fills
		t.Fatalf("expected at least 4 trade entries, got %d", len(done.Trades))
	}
	if ob.GetOrder("ice1") != nil {
		t.Fatal("iceberg should be fully consumed")
	}
}

// §18.9: Time priority — refill moves iceberg to back.
func TestIceberg_RefillMovesToBack(t *testing.T) {
	ob := newOB()
	// Place iceberg first, then a regular order behind it at same price.
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Sell, d("6"), d("3"), d("100"), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("reg1", matchingo.Sell, d("2"), d("100"), matchingo.GTC, ""))

	// exhaust first iceberg peak → refill moves ice1 to back
	done, err := ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}

	// First 3 units should come from ice1 peak, then 2 from reg1 (which is now first),
	// leaving ice1's refilled peak still in the book.
	makerFills := map[string]bool{}
	for _, tr := range done.Trades {
		makerFills[tr.OrderID] = true
	}
	if !makerFills["reg1"] {
		t.Fatal("reg1 should have filled before refilled ice1 peak")
	}
	if ob.GetOrder("ice1") == nil {
		t.Fatal("ice1 should still be in book with its refilled peak")
	}
}

// §18.9: IOC iceberg taker — no matching orders, immediately canceled.
func TestIceberg_IOCTakerNoMatch(t *testing.T) {
	ob := newOB()
	done, err := ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Buy, d("10"), d("3"), d("100"), matchingo.IOC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Canceled) == 0 {
		t.Fatal("IOC iceberg with no match should be canceled")
	}
	if ob.GetOrder("ice1") != nil {
		t.Fatal("IOC iceberg should not rest in book")
	}
}

// §18.9: IOC iceberg taker — fills available, cancels remainder.
func TestIceberg_IOCTakerPartialFill(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, d("2"), d("100"), matchingo.GTC, ""))

	done, err := ob.Process(matchingo.NewIcebergOrder("b1", matchingo.Buy, d("10"), d("3"), matchingo.FromInt(100), matchingo.IOC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) == 0 {
		t.Fatal("expected trades")
	}
	if len(done.Canceled) == 0 {
		t.Fatal("IOC iceberg remainder should be canceled")
	}
	if ob.GetOrder("b1") != nil {
		t.Fatal("IOC iceberg should not rest in book")
	}
}

// §18.9: FOK iceberg taker — volume insufficient for total amount → canceled.
func TestIceberg_FOKInsufficientVolume(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))

	// total=10 but only 5 available → FOK must cancel
	done, err := ob.Process(matchingo.NewIcebergOrder("b1", matchingo.Buy, d("10"), d("3"), d("100"), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) != 0 {
		t.Fatal("FOK iceberg with insufficient volume must not trade")
	}
	if len(done.Canceled) == 0 {
		t.Fatal("FOK iceberg should be canceled")
	}
}

// §18.9: FOK iceberg taker — checks total amount, not just peak.
func TestIceberg_FOKChecksTotalAmount(t *testing.T) {
	ob := newOB()
	// volume equals total (10), peak is 3 — FOK should pass
	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, d("10"), d("100"), matchingo.GTC, ""))

	done, err := ob.Process(matchingo.NewIcebergOrder("b1", matchingo.Buy, d("10"), d("3"), d("100"), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) == 0 {
		t.Fatal("FOK iceberg with sufficient volume should fill")
	}
}

// §18.9/§18.13: Iceberg survives snapshot round-trip — peak and hidden amount both preserved, refills correctly.
func TestIceberg_SnapshotRoundTrip(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewIcebergOrder("ice1", matchingo.Sell, d("9"), d("3"), d("100"), matchingo.GTC, ""))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	o := ob2.GetOrder("ice1")
	if o == nil {
		t.Fatal("iceberg missing after restore")
	}
	if !o.PeakQty().Equal(d("3")) {
		t.Fatalf("expected peak 3, got %s", o.PeakQty())
	}
	if !o.HiddenQty().Equal(d("6")) {
		t.Fatalf("expected hidden 6, got %s", o.HiddenQty())
	}

	// Verify refill works after restore.
	ob2.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, d("3"), d("100"), matchingo.GTC, ""))
	o = ob2.GetOrder("ice1")
	if o == nil {
		t.Fatal("iceberg should still be in book after first peak fills post-restore")
	}
	if !o.Quantity().Equal(d("3")) {
		t.Fatalf("expected refilled peak 3, got %s", o.Quantity())
	}
}
