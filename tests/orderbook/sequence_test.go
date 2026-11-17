package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestSequence_IncrementOnProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()

	if ob.Sequence() != 0 {
		t.Fatal("sequence should start at 0")
	}

	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(100), matchingo.GTC, ""))
	if ob.Sequence() != 1 {
		t.Fatalf("expected sequence 1, got %d", ob.Sequence())
	}

	ob.Process(matchingo.NewLimitOrder("s2", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(101), matchingo.GTC, ""))
	if ob.Sequence() != 2 {
		t.Fatalf("expected sequence 2, got %d", ob.Sequence())
	}
}

func TestSequence_DoneCarriesSequence(t *testing.T) {
	ob := matchingo.NewOrderBook()

	done, _ := ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	if done.Sequence != 1 {
		t.Fatalf("done.Sequence should be 1, got %d", done.Sequence)
	}

	done2, _ := ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	if done2.Sequence != 2 {
		t.Fatalf("done2.Sequence should be 2, got %d", done2.Sequence)
	}
}

func TestSequence_InvalidProcessDoesNotIncrement(t *testing.T) {
	ob := matchingo.NewOrderBook()
	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(100), matchingo.GTC, ""))

	_, err := ob.Process(nil)
	if err == nil {
		t.Fatal("expected error for nil order")
	}
	if ob.Sequence() != 1 {
		t.Fatalf("sequence should not increment on error, got %d", ob.Sequence())
	}
}

func TestSequence_SnapshotRoundTrip(t *testing.T) {
	ob := matchingo.NewOrderBook()
	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("s2", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(101), matchingo.GTC, ""))

	snap := ob.Dump()
	if snap.Sequence != 2 {
		t.Fatalf("snapshot sequence should be 2, got %d", snap.Sequence)
	}

	ob2 := matchingo.NewOrderBook()
	if err := ob2.Restore(snap); err != nil {
		t.Fatal(err)
	}
	if ob2.Sequence() != 2 {
		t.Fatalf("restored sequence should be 2, got %d", ob2.Sequence())
	}

	done, _ := ob2.Process(matchingo.NewLimitOrder("s3", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(102), matchingo.GTC, ""))
	if done.Sequence != 3 {
		t.Fatalf("post-restore sequence should resume from 3, got %d", done.Sequence)
	}
}

func TestSequence_ChannelEventCarriesSequence(t *testing.T) {
	ob := matchingo.NewOrderBook()
	ch := make(chan matchingo.Event, 16)
	ob.SetEventChannel(ch, true)

	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))

	ob.SetEmitter(nil)
	close(ch)

	var prev uint64
	for e := range ch {
		if e.Sequence < prev {
			t.Fatalf("event sequence went backwards: %d after %d", e.Sequence, prev)
		}
		prev = e.Sequence
	}
	if prev == 0 {
		t.Fatal("no events received")
	}
}

func TestSequence_TradeEventDataSequence(t *testing.T) {
	ob := matchingo.NewOrderBook()
	ch := make(chan matchingo.Event, 16)
	ob.SetEventChannel(ch, true)

	ob.Process(matchingo.NewLimitOrder("s1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("b1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))

	ob.SetEmitter(nil)
	close(ch)

	for e := range ch {
		if e.Type == matchingo.EventTrade {
			d := e.Data.(matchingo.TradeEventData)
			if d.Sequence == 0 {
				t.Fatal("TradeEventData.Sequence should be non-zero")
			}
		}
	}
}
