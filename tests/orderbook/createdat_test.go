package orderbook

import (
	"testing"
	"time"

	"github.com/gonevo/matchingo"
)

func TestSetCreatedAt_KeptAndSkipsClock(t *testing.T) {
	reads := 0
	ob := matchingo.NewOrderBook(matchingo.WithClock(func() time.Time { reads++; return time.Unix(1, 0).UTC() }))

	ts := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	o := matchingo.NewLimitOrder("a", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), "", "")
	o.SetCreatedAt(ts)
	if _, err := ob.Process(o); err != nil {
		t.Fatal(err)
	}
	if !o.CreatedAt().Equal(ts) || reads != 0 {
		t.Fatalf("createdAt=%v clockReads=%d, want %v and 0", o.CreatedAt(), reads, ts)
	}

	plain := matchingo.NewLimitOrder("b", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(10), "", "")
	if _, err := ob.Process(plain); err != nil {
		t.Fatal(err)
	}
	if !plain.CreatedAt().Equal(time.Unix(1, 0)) {
		t.Fatalf("unstamped order should get the book clock, got %v", plain.CreatedAt())
	}
}
