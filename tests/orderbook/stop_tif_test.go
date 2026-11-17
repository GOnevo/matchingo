package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// helper: submit a matching pair to trigger a trade at the given price.
func triggerTradeAt(t *testing.T, ob *matchingo.OrderBook, price matchingo.Decimal) *matchingo.Done {
	t.Helper()
	ob.Process(matchingo.NewLimitOrder("tm-"+price.String(), matchingo.Sell, d("1"), price, matchingo.GTC, ""))
	done, err := ob.Process(matchingo.NewLimitOrder("tt-"+price.String(), matchingo.Buy, d("1"), price, matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	return done
}

// §18.6: STOP-LIMIT FOK — activated with insufficient book → fully canceled.
func TestStopTIF_FOKActivatedInsufficientBook(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("2"), d("100"), matchingo.GTC, ""))

	stop1 := matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("10"), d("100"), d("90"), "")
	stop1.SetTIF(matchingo.FOK)
	ob.Process(stop1)

	done := triggerTradeAt(t, ob, d("90"))
	if len(done.Activations) == 0 {
		t.Fatal("expected stop1 activation in result")
	}
	act := done.Activations[0]
	if len(act.Trades) != 0 {
		t.Fatal("FOK with insufficient volume must not trade")
	}
	if len(act.Canceled) == 0 {
		t.Fatal("FOK with insufficient volume must be canceled")
	}
}

// §18.6: STOP-LIMIT FOK — activated with enough book → fully fills.
func TestStopTIF_FOKActivatedSufficientBook(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("10"), d("100"), matchingo.GTC, ""))

	stop1 := matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("5"), d("100"), d("90"), "")
	stop1.SetTIF(matchingo.FOK)
	ob.Process(stop1)

	done := triggerTradeAt(t, ob, d("90"))
	if len(done.Activations) == 0 {
		t.Fatal("expected stop1 activation")
	}
	act := done.Activations[0]
	if len(act.Trades) == 0 {
		t.Fatal("FOK with sufficient volume must fill")
	}
	if len(act.Canceled) != 0 {
		t.Fatal("FOK fully filled should not have cancel")
	}
}

// §18.6: STOP-LIMIT IOC — activated with insufficient book → partial fill, remainder canceled.
func TestStopTIF_IOCActivatedPartialFill(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("3"), d("100"), matchingo.GTC, ""))

	stop1 := matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("10"), d("100"), d("90"), "")
	stop1.SetTIF(matchingo.IOC)
	ob.Process(stop1)

	done := triggerTradeAt(t, ob, d("90"))
	if len(done.Activations) == 0 {
		t.Fatal("expected stop1 activation")
	}
	act := done.Activations[0]
	if len(act.Trades) == 0 {
		t.Fatal("IOC should partially fill")
	}
	if len(act.Canceled) == 0 {
		t.Fatal("IOC remainder should be canceled")
	}
}

// §18.6: STOP-LIMIT IOC — activated and fully fills → no cancel in result.
func TestStopTIF_IOCActivatedFullyFills(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("10"), d("100"), matchingo.GTC, ""))

	stop1 := matchingo.NewStopLimitOrder("stop1", matchingo.Buy, d("5"), d("100"), d("90"), "")
	stop1.SetTIF(matchingo.IOC)
	ob.Process(stop1)

	done := triggerTradeAt(t, ob, d("90"))
	if len(done.Activations) == 0 {
		t.Fatal("expected stop1 activation")
	}
	act := done.Activations[0]
	if len(act.Trades) == 0 {
		t.Fatal("IOC should fill")
	}
	if len(act.Canceled) != 0 {
		t.Fatal("fully filled IOC should not have cancel entry")
	}
}
