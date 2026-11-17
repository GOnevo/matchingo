package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// §18.5: MARKET BUY quote — quote spent never exceeds budget.
// price=5, quote=20 → buys 4 base (exact); ceiling charge = 4*5=20.
func TestMarketQuote_SpentDoesNotExceedBudget(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("10"), d("5"), matchingo.GTC, ""))

	done, err := ob.Process(matchingo.NewMarketQuoteOrder("mq1", matchingo.Buy, d("20")))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) == 0 {
		t.Fatal("expected trades")
	}
	// quote spent must be ≤ budget (20)
	if done.Processed.GreaterThan(d("20")) {
		t.Fatalf("spent %s exceeds quote budget 20", done.Processed)
	}
}

// §18.5: MARKET BUY quote — budget exceeds book → partial fill, dust canceled.
func TestMarketQuote_BudgetExceedsBook(t *testing.T) {
	ob := newOB()
	// only 2 base units available at price 5
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("2"), d("5"), matchingo.GTC, ""))

	done, err := ob.Process(matchingo.NewMarketQuoteOrder("mq1", matchingo.Buy, d("100")))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Trades) == 0 {
		t.Fatal("expected trades")
	}
	// spent must be ≤ 100 and > 0
	if done.Processed.Equal(matchingo.Zero) {
		t.Fatal("expected non-zero processed amount")
	}
	if done.Processed.GreaterThan(d("100")) {
		t.Fatalf("spent %s exceeds budget 100", done.Processed)
	}
}

// §18.5: MARKET BUY quote — budget exceeds remaining book, order is fully consumed.
// price=5, budget=100 but only 2 base available → partial fill, no open order remains.
func TestMarketQuote_NoOpenOrderAfterPartialFill(t *testing.T) {
	ob := newOB()
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, d("2"), d("5"), matchingo.GTC, ""))

	done, err := ob.Process(matchingo.NewMarketQuoteOrder("mq1", matchingo.Buy, d("100")))
	if err != nil {
		t.Fatal(err)
	}
	// MARKET order must not remain in book
	if ob.GetOrder("mq1") != nil {
		t.Fatal("MARKET quote order should not remain in book")
	}
	if done.Processed.Equal(matchingo.Zero) {
		t.Fatal("expected some quote processed")
	}
}
