package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestStopMarket_ActivatesAndSweepsBook(t *testing.T) {
	book := matchingo.NewOrderBook()
	book.Process(matchingo.NewStopMarketOrder("stop", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(10), ""))
	book.Process(matchingo.NewLimitOrder("ask10", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))
	book.Process(matchingo.NewLimitOrder("ask12", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(12), matchingo.GTC, ""))
	book.Process(matchingo.NewLimitOrder("ask15", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(15), matchingo.GTC, ""))

	done, err := book.Process(matchingo.NewMarketOrder("trigger", matchingo.Buy, matchingo.FromInt(1)))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Activated) != 1 || done.Activated[0] != "stop" {
		t.Fatalf("activated = %v, want [stop]", done.Activated)
	}
	act := done.Activations[0]
	if !act.Processed.Equal(matchingo.FromInt(3)) || !act.Left.Equal(Zero) {
		t.Fatalf("processed=%s left=%s, want 3/0", act.Processed, act.Left)
	}
	// no price cap: swept through 12 and into 15
	if tr := act.GetTradeOrder("ask15"); tr == nil || !tr.Quantity.Equal(matchingo.FromInt(1)) {
		t.Fatal("expected 1 unit taken from ask15")
	}
	if book.GetOrder("stop") != nil {
		t.Fatal("market-activated stop must not remain in the book")
	}
}

func TestStopMarket_RemainderCanceledWhenBookThin(t *testing.T) {
	book := matchingo.NewOrderBook()
	book.Process(matchingo.NewStopMarketOrder("stop", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(10), ""))
	book.Process(matchingo.NewLimitOrder("bid10", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))
	book.Process(matchingo.NewLimitOrder("bid9", matchingo.Buy, matchingo.FromInt(2), matchingo.FromInt(9), matchingo.GTC, ""))

	done, _ := book.Process(matchingo.NewMarketOrder("trigger", matchingo.Sell, matchingo.FromInt(1)))
	if len(done.Activations) != 1 {
		t.Fatalf("activations = %d, want 1", len(done.Activations))
	}
	act := done.Activations[0]
	if !act.Processed.Equal(matchingo.FromInt(2)) || !act.Left.Equal(matchingo.FromInt(3)) {
		t.Fatalf("processed=%s left=%s, want 2/3", act.Processed, act.Left)
	}
	if len(act.Canceled) != 1 || act.Canceled[0] != "stop" || book.GetOrder("stop") != nil {
		t.Fatal("unfilled remainder must be canceled and not rest")
	}
}

func TestStopMarket_NotTriggeredBelowStop(t *testing.T) {
	book := matchingo.NewOrderBook()
	book.Process(matchingo.NewStopMarketOrder("stop", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), ""))
	book.Process(matchingo.NewLimitOrder("ask", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(9), matchingo.GTC, ""))
	done, _ := book.Process(matchingo.NewMarketOrder("m", matchingo.Buy, matchingo.FromInt(1)))
	if len(done.Activated) != 0 || book.GetOrder("stop") == nil {
		t.Fatal("stop must stay dormant below its stop price")
	}
}

func TestStopMarket_CancelModifyAndSnapshot(t *testing.T) {
	book := matchingo.NewOrderBook()
	book.Process(matchingo.NewStopMarketOrder("stop", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), ""))
	if err := book.ModifyStop("stop", matchingo.FromInt(11), matchingo.FromInt(99), matchingo.FromInt(2)); err != nil {
		t.Fatal(err)
	}
	o := book.GetOrder("stop")
	if !o.StopPrice().Equal(matchingo.FromInt(11)) || !o.Quantity().Equal(matchingo.FromInt(2)) || !o.Price().Equal(Zero) {
		t.Fatalf("modify result wrong: %v stop=%s", o, o.StopPrice())
	}

	restored := matchingo.NewOrderBook()
	if err := restored.Restore(book.Dump()); err != nil {
		t.Fatal(err)
	}
	restored.Process(matchingo.NewLimitOrder("ask", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(11), matchingo.GTC, ""))
	done, _ := restored.Process(matchingo.NewMarketOrder("m", matchingo.Buy, matchingo.FromInt(1)))
	if len(done.Activated) != 1 || !done.Activations[0].Processed.Equal(matchingo.FromInt(2)) {
		t.Fatalf("restored stop-market did not activate: %v", done.Activated)
	}

	if book.CancelOrder("stop") == nil || book.GetOrder("stop") != nil {
		t.Fatal("cancel failed")
	}
}

func TestStopMarket_OCO(t *testing.T) {
	// stop-market fills -> sibling limit canceled
	book := matchingo.NewOrderBook()
	book.Process(matchingo.NewLimitOrder("tp", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(20), matchingo.GTC, "sl"))
	book.Process(matchingo.NewStopMarketOrder("sl", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(10), "tp"))
	book.Process(matchingo.NewLimitOrder("bid", matchingo.Buy, matchingo.FromInt(2), matchingo.FromInt(10), matchingo.GTC, ""))
	done, _ := book.Process(matchingo.NewMarketOrder("trig", matchingo.Sell, matchingo.FromInt(1)))
	if len(done.Activated) != 1 || book.GetOrder("tp") != nil {
		t.Fatalf("sibling should be canceled; activated=%v", done.Activated)
	}

	// sibling fills first -> stop-market canceled
	book = matchingo.NewOrderBook()
	book.Process(matchingo.NewLimitOrder("tp", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(20), matchingo.GTC, "sl"))
	book.Process(matchingo.NewStopMarketOrder("sl", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(10), "tp"))
	book.Process(matchingo.NewMarketOrder("buy", matchingo.Buy, matchingo.FromInt(1)))
	if book.GetOrder("sl") != nil || book.Stop.Len() != 0 {
		t.Fatal("stop-market should be canceled when sibling fills")
	}
}

func TestStopMarket_ConstructorPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"qty":  func() { matchingo.NewStopMarketOrder("a", matchingo.Buy, Zero, matchingo.FromInt(1), "") },
		"stop": func() { matchingo.NewStopMarketOrder("a", matchingo.Buy, matchingo.FromInt(1), Zero, "") },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("%s: expected panic", name)
				}
			}()
			f()
		}()
	}
}

var Zero = matchingo.Zero
