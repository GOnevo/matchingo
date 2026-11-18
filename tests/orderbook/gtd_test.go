package orderbook

import (
	"errors"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
)

func gtdBook() (*matchingo.OrderBook, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return matchingo.NewOrderBook(matchingo.WithClock(func() time.Time { return now })), &now
}

func TestGTD_ExpiresOnNextProcess(t *testing.T) {
	b, now := gtdBook()
	b.Process(matchingo.NewGTDOrder("g", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(10), now.Add(time.Minute), ""))
	b.Process(matchingo.NewLimitOrder("keep", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(11), matchingo.GTC, ""))

	*now = now.Add(time.Minute) // expiry is inclusive
	done, err := b.Process(matchingo.NewMarketOrder("m", matchingo.Buy, matchingo.FromInt(1)))
	if err != nil {
		t.Fatal(err)
	}
	if b.GetOrder("g") != nil {
		t.Fatal("expired order still on book")
	}
	if len(done.Canceled) != 1 || done.Canceled[0] != "g" {
		t.Fatalf("canceled = %v, want [g]", done.Canceled)
	}
	if tr := done.GetTradeOrder("keep"); tr == nil || !tr.Price.Equal(matchingo.FromInt(11)) {
		t.Fatal("market order must not match the expired order")
	}
}

func TestGTD_LiveOrderMatchesBeforeExpiry(t *testing.T) {
	b, now := gtdBook()
	b.Process(matchingo.NewGTDOrder("g", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(10), now.Add(time.Minute), ""))
	done, _ := b.Process(matchingo.NewMarketOrder("m", matchingo.Buy, matchingo.FromInt(1)))
	if done.GetTradeOrder("g") == nil {
		t.Fatal("unexpired GTD order should match")
	}
}

func TestGTD_ExpireOrdersExplicit(t *testing.T) {
	b, now := gtdBook()
	b.Process(matchingo.NewGTDOrder("late", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(9), now.Add(2*time.Minute), ""))
	b.Process(matchingo.NewGTDOrder("early", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(8), now.Add(time.Minute), ""))
	exp := b.ExpireOrders(now.Add(90 * time.Second))
	if len(exp) != 1 || exp[0].ID() != "early" || b.GetOrder("late") == nil {
		t.Fatalf("expired = %v", exp)
	}
	exp = b.ExpireOrders(now.Add(time.Hour))
	if len(exp) != 1 || exp[0].ID() != "late" || b.BestBid() != nil {
		t.Fatal("second sweep should expire the rest")
	}
}

func TestGTD_RejectsAlreadyExpired(t *testing.T) {
	b, now := gtdBook()
	_, err := b.Process(matchingo.NewGTDOrder("g", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), now.Add(-time.Second), ""))
	if !errors.Is(err, matchingo.ErrOrderExpired) {
		t.Fatalf("err = %v, want ErrOrderExpired", err)
	}
}

func TestGTD_SnapshotKeepsExpiry(t *testing.T) {
	b, now := gtdBook()
	b.Process(matchingo.NewGTDOrder("g", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), now.Add(time.Minute), ""))
	r := matchingo.NewOrderBook(matchingo.WithClock(func() time.Time { return now.Add(time.Hour) }))
	if err := r.Restore(b.Dump()); err != nil {
		t.Fatal(err)
	}
	r.Process(matchingo.NewLimitOrder("x", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(5), matchingo.GTC, ""))
	if r.GetOrder("g") != nil {
		t.Fatal("restored GTD order should expire")
	}
}

func TestTTL_ExpiresRelativeToSubmission(t *testing.T) {
	b, now := gtdBook()
	o := matchingo.NewLimitOrder("t", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), matchingo.GTC, "")
	o.SetTTL(time.Minute)
	if _, err := b.Process(o); err != nil {
		t.Fatal(err)
	}
	if !o.ExpiresAt().Equal(now.Add(time.Minute)) || o.TIF() != matchingo.GTD {
		t.Fatalf("expiresAt=%v tif=%v", o.ExpiresAt(), o.TIF())
	}
	*now = now.Add(time.Minute)
	b.ExpireOrders(*now)
	if b.GetOrder("t") != nil {
		t.Fatal("TTL order should have expired")
	}
}

func TestTTL_StopOrderAndRejections(t *testing.T) {
	b, now := gtdBook()
	s := matchingo.NewStopLimitOrder("s", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(11), matchingo.FromInt(10), "")
	s.SetTTL(time.Minute)
	b.Process(s)
	exp := b.ExpireOrders(now.Add(time.Minute))
	if len(exp) != 1 || exp[0].ID() != "s" || b.Stop.Len() != 0 {
		t.Fatal("stop order should expire and leave the stop book")
	}

	for name, o := range map[string]*matchingo.Order{
		"ioc":    matchingo.NewLimitOrder("i", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(1), matchingo.IOC, ""),
		"market": matchingo.NewMarketOrder("m", matchingo.Buy, matchingo.FromInt(1)),
	} {
		o.SetTTL(time.Minute)
		if _, err := b.Process(o); !errors.Is(err, matchingo.ErrInvalidTif) {
			t.Fatalf("%s: err = %v, want ErrInvalidTif", name, err)
		}
	}
	defer func() {
		if recover() == nil {
			t.Fatal("non-positive TTL must panic")
		}
	}()
	o := matchingo.NewMarketOrder("z", matchingo.Buy, matchingo.FromInt(1))
	o.SetTTL(0)
}
