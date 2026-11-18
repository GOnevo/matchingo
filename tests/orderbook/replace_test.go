package orderbook

import (
	"errors"
	"testing"

	"github.com/gonevo/matchingo"
)

func TestReplace_SwapsAndLosesPriority(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewLimitOrder("a", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))
	b.Process(matchingo.NewLimitOrder("b", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))

	done, err := b.Replace("a", matchingo.NewLimitOrder("a", matchingo.Buy, matchingo.FromInt(2), matchingo.FromInt(10), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Canceled) != 1 || done.Canceled[0] != "a" || !done.Stored {
		t.Fatalf("canceled=%v stored=%v", done.Canceled, done.Stored)
	}
	d, _ := b.Process(matchingo.NewMarketOrder("m", matchingo.Sell, matchingo.FromInt(1)))
	if d.GetTradeOrder("b") == nil || d.GetTradeOrder("a") != nil {
		t.Fatal("replaced order must queue behind b")
	}
}

func TestReplace_MatchesImmediately(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewLimitOrder("ask", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(11), matchingo.GTC, ""))
	b.Process(matchingo.NewLimitOrder("bid", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))
	done, err := b.Replace("bid", matchingo.NewLimitOrder("bid2", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(11), matchingo.GTC, ""))
	if err != nil || !done.Processed.Equal(matchingo.FromInt(1)) || b.GetOrder("bid") != nil {
		t.Fatalf("err=%v done=%v", err, done)
	}
}

func TestReplace_FailureLeavesOldUntouched(t *testing.T) {
	b := matchingo.NewOrderBook(matchingo.WithPrecision(2, 2))
	b.Process(matchingo.NewLimitOrder("a", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, ""))
	b.Process(matchingo.NewLimitOrder("other", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(9), matchingo.GTC, ""))
	seq := b.Sequence()

	bad := matchingo.NewLimitOrder("a", matchingo.Buy, matchingo.FromFloat(1.005), matchingo.FromInt(10), matchingo.GTC, "")
	if _, err := b.Replace("a", bad); !errors.Is(err, matchingo.ErrQuantityPrecision) {
		t.Fatalf("err = %v, want precision error", err)
	}
	if _, err := b.Replace("a", matchingo.NewLimitOrder("other", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(8), matchingo.GTC, "")); !errors.Is(err, matchingo.ErrOrderExists) {
		t.Fatalf("err = %v, want ErrOrderExists", err)
	}
	if _, err := b.Replace("nope", matchingo.NewLimitOrder("x", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(8), matchingo.GTC, "")); !errors.Is(err, matchingo.ErrOrderNotFound) {
		t.Fatalf("err = %v, want ErrOrderNotFound", err)
	}
	if b.GetOrder("a") == nil || b.Sequence() != seq {
		t.Fatal("old order must be intact after a failed replace")
	}
}

func TestReplace_StopOrder(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewStopLimitOrder("s", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(11), matchingo.FromInt(10), ""))
	if _, err := b.Replace("s", matchingo.NewStopMarketOrder("s", matchingo.Buy, matchingo.FromInt(2), matchingo.FromInt(12), "")); err != nil {
		t.Fatal(err)
	}
	o := b.GetOrder("s")
	if o == nil || !o.StopPrice().Equal(matchingo.FromInt(12)) || b.Stop.Len() != 1 {
		t.Fatalf("stop not replaced: %v", o)
	}
}
