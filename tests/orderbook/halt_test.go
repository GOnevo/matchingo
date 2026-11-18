package orderbook

import (
	"errors"
	"testing"

	"github.com/gonevo/matchingo"
)

func limit(id string, side matchingo.Side, qty, price int) *matchingo.Order {
	return matchingo.NewLimitOrder(id, side, matchingo.FromInt(qty), matchingo.FromInt(price), matchingo.GTC, "")
}

func TestHalt_RejectsNewWorkKeepsBookAndAllowsCancel(t *testing.T) {
	em := &testEmitter{}
	b := matchingo.NewOrderBook(matchingo.WithEmitter(em))
	b.Process(limit("bid", matchingo.Buy, 1, 10))
	b.Process(matchingo.NewStopLimitOrder("stop", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(21), matchingo.FromInt(20), ""))
	seq := b.Sequence()

	b.Halt()
	if !b.IsHalted() {
		t.Fatal("IsHalted = false")
	}
	if _, err := b.Process(limit("x", matchingo.Sell, 1, 10)); !errors.Is(err, matchingo.ErrTradingHalted) {
		t.Fatalf("Process err = %v", err)
	}
	if _, err := b.Replace("bid", limit("bid", matchingo.Buy, 2, 10)); !errors.Is(err, matchingo.ErrTradingHalted) {
		t.Fatalf("Replace err = %v", err)
	}
	if _, err := b.Modify("bid", matchingo.FromInt(10), matchingo.FromInt(1)); !errors.Is(err, matchingo.ErrTradingHalted) {
		t.Fatalf("Modify err = %v", err)
	}
	if err := b.ModifyStop("stop", matchingo.FromInt(20), matchingo.FromInt(21), matchingo.FromInt(1)); !errors.Is(err, matchingo.ErrTradingHalted) {
		t.Fatalf("ModifyStop err = %v", err)
	}
	if b.GetOrder("bid") == nil || b.Sequence() != seq {
		t.Fatal("book must be intact and sequence unchanged")
	}
	if b.CancelOrder("stop") == nil {
		t.Fatal("cancel must work while halted")
	}

	b.Resume()
	if _, err := b.Process(limit("x", matchingo.Sell, 1, 10)); err != nil {
		t.Fatalf("after resume: %v", err)
	}
	ev := em.eventsOf(matchingo.EventHalt)
	if len(ev) != 2 || !ev[0].data.(matchingo.HaltEventData).Halted || ev[1].data.(matchingo.HaltEventData).Halted {
		t.Fatalf("halt events = %v", ev)
	}
	b.Resume() // idempotent: no extra event
	if len(em.eventsOf(matchingo.EventHalt)) != 2 {
		t.Fatal("redundant Resume emitted an event")
	}
}

func tradeAt(t *testing.T, b *matchingo.OrderBook, id string, price int) {
	t.Helper()
	b.Process(limit(id+"a", matchingo.Sell, 1, price))
	if _, err := b.Process(limit(id+"b", matchingo.Buy, 1, price)); err != nil {
		t.Fatalf("trade %s: %v", id, err)
	}
}

func TestCircuitBreaker_TripsOnLargeMove(t *testing.T) {
	em := &testEmitter{}
	b := matchingo.NewOrderBook(matchingo.WithCircuitBreaker(matchingo.FromInt(10)), matchingo.WithEmitter(em))
	tradeAt(t, b, "t1", 100) // sets reference
	tradeAt(t, b, "t2", 110) // exactly 10%: not above the threshold
	if b.IsHalted() {
		t.Fatal("a move equal to the threshold must not trip")
	}
	tradeAt(t, b, "t3", 90) // ~18% down from 110
	if !b.IsHalted() {
		t.Fatal("breaker should have tripped")
	}
	if ev := em.eventsOf(matchingo.EventHalt); len(ev) != 1 || !ev[0].data.(matchingo.HaltEventData).Auto {
		t.Fatalf("halt events = %v", ev)
	}
	if _, err := b.Process(limit("n", matchingo.Buy, 1, 90)); !errors.Is(err, matchingo.ErrTradingHalted) {
		t.Fatalf("err = %v", err)
	}
	b.Resume()
	tradeAt(t, b, "t4", 92) // reference is now 90
	if b.IsHalted() {
		t.Fatal("small move after resume must not trip")
	}
}

func TestCircuitBreaker_OffByDefaultAndSnapshot(t *testing.T) {
	b := matchingo.NewOrderBook()
	tradeAt(t, b, "t1", 100)
	tradeAt(t, b, "t2", 1000)
	if b.IsHalted() {
		t.Fatal("no breaker configured")
	}
	b.Halt()
	r := matchingo.NewOrderBook()
	if err := r.Restore(b.Dump()); err != nil {
		t.Fatal(err)
	}
	if !r.IsHalted() {
		t.Fatal("halt state must survive Dump/Restore")
	}
}
