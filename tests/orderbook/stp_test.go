package orderbook

import (
	"errors"
	"testing"

	"github.com/gonevo/matchingo"
)

func stpBook() *matchingo.OrderBook {
	b := matchingo.NewOrderBook()
	own := matchingo.NewLimitOrder("own", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(10), matchingo.GTC, "")
	own.SetSTP("alice", matchingo.STPNone)
	b.Process(own)
	b.Process(matchingo.NewLimitOrder("other", matchingo.Sell, matchingo.FromInt(2), matchingo.FromInt(11), matchingo.GTC, ""))
	return b
}

func buyer(mode matchingo.STPMode, qty int) *matchingo.Order {
	o := matchingo.NewLimitOrder("buy", matchingo.Buy, matchingo.FromInt(qty), matchingo.FromInt(11), matchingo.GTC, "")
	o.SetSTP("alice", mode)
	return o
}

func TestSTP_NoModeTradesNormally(t *testing.T) {
	b := stpBook()
	done, _ := b.Process(buyer(matchingo.STPNone, 1))
	if done.GetTradeOrder("own") == nil {
		t.Fatal("STPNone must trade with own order")
	}
}

func TestSTP_DifferentParticipantTrades(t *testing.T) {
	b := stpBook()
	o := buyer(matchingo.STPCancelNewest, 1)
	o.SetSTP("bob", matchingo.STPCancelNewest)
	done, _ := b.Process(o)
	if done.GetTradeOrder("own") == nil {
		t.Fatal("different participants must trade")
	}
}

func TestSTP_CancelNewest(t *testing.T) {
	b := stpBook()
	done, _ := b.Process(buyer(matchingo.STPCancelNewest, 3))
	if len(done.Trades) != 0 || len(done.Canceled) != 1 || done.Canceled[0] != "buy" || done.Stored {
		t.Fatalf("trades=%v canceled=%v stored=%v", done.Trades, done.Canceled, done.Stored)
	}
	if b.GetOrder("own") == nil || b.GetOrder("buy") != nil {
		t.Fatal("resting order must stay, incoming must not rest")
	}
}

func TestSTP_CancelOldestKeepsMatching(t *testing.T) {
	b := stpBook()
	done, _ := b.Process(buyer(matchingo.STPCancelOldest, 1))
	if b.GetOrder("own") != nil || len(done.Canceled) != 1 || done.Canceled[0] != "own" {
		t.Fatalf("own should be canceled: %v", done.Canceled)
	}
	if tr := done.GetTradeOrder("other"); tr == nil || !tr.Quantity.Equal(matchingo.FromInt(1)) {
		t.Fatal("incoming should continue to trade with the other participant")
	}
}

func TestSTP_CancelBoth(t *testing.T) {
	b := stpBook()
	done, _ := b.Process(buyer(matchingo.STPCancelBoth, 1))
	if b.GetOrder("own") != nil || b.GetOrder("buy") != nil || len(done.Canceled) != 2 || len(done.Trades) != 0 {
		t.Fatalf("canceled=%v trades=%v", done.Canceled, done.Trades)
	}
}

func TestSTP_DecrementPartial(t *testing.T) {
	b := stpBook()
	done, _ := b.Process(buyer(matchingo.STPDecrement, 1))
	own := b.GetOrder("own")
	if own == nil || !own.Quantity().Equal(matchingo.FromInt(1)) {
		t.Fatalf("own should shrink to 1: %v", own)
	}
	if len(done.Trades) != 0 || len(done.Canceled) != 1 || done.Canceled[0] != "buy" || b.GetOrder("buy") != nil {
		t.Fatalf("incoming fully decremented must be canceled: %v %v", done.Trades, done.Canceled)
	}
}

func TestSTP_DecrementThenTrade(t *testing.T) {
	b := stpBook()
	done, _ := b.Process(buyer(matchingo.STPDecrement, 3))
	if b.GetOrder("own") != nil {
		t.Fatal("own (2) fully decremented")
	}
	if !done.Processed.Equal(matchingo.FromInt(1)) || !done.Left.Equal(Zero) {
		t.Fatalf("processed=%s left=%s, want 1/0", done.Processed, done.Left)
	}
	if tr := done.GetTradeOrder("other"); tr == nil || !tr.Quantity.Equal(matchingo.FromInt(1)) {
		t.Fatal("remaining 1 should trade with other")
	}
}

func TestSTP_MarketTaker(t *testing.T) {
	b := stpBook()
	m := matchingo.NewMarketOrder("m", matchingo.Buy, matchingo.FromInt(3))
	m.SetSTP("alice", matchingo.STPCancelNewest)
	done, _ := b.Process(m)
	if len(done.Trades) != 0 || len(done.Canceled) != 1 || b.GetOrder("own") == nil {
		t.Fatalf("trades=%v canceled=%v", done.Trades, done.Canceled)
	}
}

func TestSTP_QuoteMarketCancelOldestDoesNotStall(t *testing.T) {
	b := stpBook()
	m := matchingo.NewMarketQuoteOrder("m", matchingo.Buy, matchingo.FromInt(22))
	m.SetSTP("alice", matchingo.STPCancelOldest)
	done, _ := b.Process(m)
	if b.GetOrder("own") != nil || done.GetTradeOrder("other") == nil {
		t.Fatalf("own should be canceled and other traded: %v", done)
	}
}

func TestSTP_RejectsFOKAndInvalidMode(t *testing.T) {
	b := stpBook()
	o := matchingo.NewLimitOrder("f", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(11), matchingo.FOK, "")
	o.SetSTP("alice", matchingo.STPCancelNewest)
	if _, err := b.Process(o); !errors.Is(err, matchingo.ErrInvalidSTP) {
		t.Fatalf("err = %v, want ErrInvalidSTP", err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("invalid mode must panic")
		}
	}()
	o.SetSTP("alice", matchingo.STPMode(99))
}

func TestSTP_SnapshotKeepsParticipant(t *testing.T) {
	b := stpBook()
	r := matchingo.NewOrderBook()
	if err := r.Restore(b.Dump()); err != nil {
		t.Fatal(err)
	}
	done, _ := r.Process(buyer(matchingo.STPCancelNewest, 1))
	if len(done.Trades) != 0 || r.GetOrder("own") == nil {
		t.Fatal("restored resting order lost its participant")
	}
}
