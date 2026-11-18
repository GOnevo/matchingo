package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// trade executes one unit at price using a resting bid/ask pair.
func trade(t *testing.T, b *matchingo.OrderBook, id string, price int) *matchingo.Done {
	t.Helper()
	b.Process(matchingo.NewLimitOrder(id+"-a", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(price), matchingo.GTC, ""))
	done, err := b.Process(matchingo.NewLimitOrder(id+"-b", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(price), matchingo.GTC, ""))
	if err != nil {
		t.Fatal(err)
	}
	return done
}

func TestTrailingStop_SellRatchetsUpOnly(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewTrailingStopOrder("ts", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(95), matchingo.FromInt(5), ""))
	want := func(v int) {
		t.Helper()
		if got := b.GetOrder("ts").StopPrice(); !got.Equal(matchingo.FromInt(v)) {
			t.Fatalf("stop = %s, want %d", got, v)
		}
	}
	trade(t, b, "t1", 110)
	want(105)
	trade(t, b, "t2", 108) // lower trade: stop must not move back, 105 < 108 so not triggered
	want(105)
	trade(t, b, "t3", 120)
	want(115)
}

func TestTrailingStop_SellActivatesAsMarket(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewTrailingStopOrder("ts", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(95), matchingo.FromInt(5), ""))
	trade(t, b, "t1", 110) // stop -> 105
	b.Process(matchingo.NewLimitOrder("bid", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(90), matchingo.GTC, ""))
	done := trade(t, b, "t2", 104)
	if len(done.Activated) != 1 || done.Activated[0] != "ts" {
		t.Fatalf("activated = %v, want [ts]", done.Activated)
	}
	if !done.Activations[0].Processed.Equal(matchingo.FromInt(1)) {
		t.Fatal("activated trailing stop should fill against the bid")
	}
}

func TestTrailingStop_BuyRatchetsDown(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewTrailingStopOrder("ts", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(120), matchingo.FromInt(5), ""))
	trade(t, b, "t1", 100)
	if got := b.GetOrder("ts").StopPrice(); !got.Equal(matchingo.FromInt(105)) {
		t.Fatalf("stop = %s, want 105", got)
	}
	trade(t, b, "t2", 102)
	if got := b.GetOrder("ts").StopPrice(); !got.Equal(matchingo.FromInt(105)) {
		t.Fatalf("stop = %s, want 105 (102+5=107 is worse)", got)
	}
}

func TestTrailingStop_Percent(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewTrailingStopPercentOrder("ts", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(50), matchingo.FromInt(10), ""))
	trade(t, b, "t1", 200)
	if got := b.GetOrder("ts").StopPrice(); !got.Equal(matchingo.FromInt(180)) {
		t.Fatalf("stop = %s, want 180", got)
	}
}

func TestTrailingStop_SnapshotKeepsTrail(t *testing.T) {
	b := matchingo.NewOrderBook()
	b.Process(matchingo.NewTrailingStopOrder("ts", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(95), matchingo.FromInt(5), ""))
	r := matchingo.NewOrderBook()
	if err := r.Restore(b.Dump()); err != nil {
		t.Fatal(err)
	}
	trade(t, r, "t1", 110)
	if got := r.GetOrder("ts").StopPrice(); !got.Equal(matchingo.FromInt(105)) {
		t.Fatalf("restored stop = %s, want 105", got)
	}
}

func TestTrailingStop_ConstructorPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"zero-trail": func() {
			matchingo.NewTrailingStopOrder("a", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(1), Zero, "")
		},
		"pct-100": func() {
			matchingo.NewTrailingStopPercentOrder("a", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(1), matchingo.FromInt(100), "")
		},
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
