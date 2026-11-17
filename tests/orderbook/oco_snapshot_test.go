package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.8: After restore, stop activates and cancels limit sibling.
func TestOCOSnapshot_StopActivatesCancelsLimitSibling(t *testing.T) {
	ob := newOB()
	// limit order resting in book, linked to a stop sibling
	ob.Process(matchingo.NewLimitOrder("lim1", matchingo.Buy, d("5"), d("99"), matchingo.GTC, "stop1"))
	ob.Process(matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("5"), d("105"), d("100"), "lim1"))

	snap := ob.Dump()
	ob2 := newOB()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}

	// verify both survived restore
	if ob2.GetOrder("lim1") == nil {
		t.Fatal("lim1 not found after restore")
	}
	if ob2.GetOrder("stop1") == nil {
		t.Fatal("stop1 not found after restore")
	}

	// add an ask that stop1 can fill against when activated
	ob2.Process(matchingo.NewLimitOrder("ask-fill", matchingo.Sell, d("5"), d("105"), matchingo.GTC, ""))
	// trigger a trade at 100 → stop1 activates → fills ask-fill → cancels lim1
	ob2.Process(matchingo.NewLimitOrder("ask-trig", matchingo.Sell, d("1"), d("100"), matchingo.GTC, ""))
	done, err := ob2.Process(matchingo.NewLimitOrder("bid-trig", matchingo.Buy, d("1"), d("100"), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Activated) == 0 {
		t.Fatal("stop1 should activate after trade at 100")
	}

	// lim1 should be canceled as the OCO sibling of stop1
	canceled := false
	checkCanceled := func(d *matchingo.Done) {
		for _, c := range d.Canceled {
			if c == "lim1" {
				canceled = true
			}
		}
	}
	checkCanceled(done)
	for _, act := range done.Activations {
		checkCanceled(act)
	}
	if !canceled {
		t.Fatal("lim1 should be canceled when stop1 activates after restore")
	}
	if ob2.GetOrder("lim1") != nil {
		t.Fatal("lim1 should be removed from book after OCO cancel")
	}
}
