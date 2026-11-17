package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.14: Cancel resting BUY limit — returns true; volume cleared.
func TestCancel_BuyLimitReturnsTrueAndClearsVolume(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, d("5"), d("100"), matchingo.GTC, ""))

	if !ob.Cancel("bid1") {
		t.Fatal("Cancel should return true for existing order")
	}
	if ob.GetOrder("bid1") != nil {
		t.Fatal("order should be removed from book")
	}
	v := ob.Level(matchingo.Buy, d("100"))
	if v != nil {
		t.Fatal("volume at 100 should be zero after cancel")
	}
}

// §18.14: Cancel resting SELL limit — returns true; volume cleared.
func TestCancel_SellLimitReturnsTrueAndClearsVolume(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("5"), d("100"), matchingo.GTC, ""))

	if !ob.Cancel("ask1") {
		t.Fatal("Cancel should return true for existing order")
	}
	v := ob.Level(matchingo.Sell, d("100"))
	if v != nil {
		t.Fatal("volume at 100 should be zero after cancel")
	}
}

// §18.14: Cancel STOP-LIMIT — returns true.
func TestCancel_StopLimitReturnsTrue(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("5"), d("105"), d("90"), ""))

	if !ob.Cancel("stop1") {
		t.Fatal("Cancel should return true for existing stop order")
	}
	if ob.GetOrder("stop1") != nil {
		t.Fatal("stop order should be removed")
	}
}

// §18.14: Cancel non-existent ID returns false.
func TestCancel_NonExistentReturnsFalse(t *testing.T) {
	ob := newOB()
	if ob.Cancel("ghost") {
		t.Fatal("Cancel non-existent should return false")
	}
}
