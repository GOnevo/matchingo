package orderbook

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/gonevo/matchingo"
)

type memWAL struct{ recs []matchingo.WALRecord }

func (m *memWAL) Append(r matchingo.WALRecord) error {
	m.recs = append(m.recs, r)
	return nil
}

// Hammer the book from many goroutines (run with -race), then prove the outcome equals
// one single-threaded execution: replaying the WAL into a plain book gives the same state.
func TestConcurrent_LinearizableUnderLoad(t *testing.T) {
	w := &memWAL{}
	c := matchingo.NewConcurrentOrderBook(matchingo.WithWAL(w))

	const workers, perWorker = 8, 300
	var wg sync.WaitGroup
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				id := fmt.Sprintf("g%d-%d", g, i)
				price := 95 + (i+g)%10
				side := matchingo.Buy
				if (i+g)%2 == 0 {
					side = matchingo.Sell
				}
				switch i % 6 {
				case 0, 1, 2:
					c.Process(matchingo.NewLimitOrder(id, side, d("2"), matchingo.FromInt(price), matchingo.GTC, ""))
				case 3:
					c.Process(matchingo.NewMarketOrder(id, side, d("1")))
				case 4:
					c.CancelOrder(fmt.Sprintf("g%d-%d", g, i-4))
				case 5:
					c.Process(matchingo.NewStopLimitOrder(id, side, d("1"), matchingo.FromInt(price), matchingo.FromInt(price), ""))
				}
				// concurrent readers
				c.BestBid()
				c.GetOrder(id)
				if i%50 == 0 {
					c.Depth()
					c.Dump()
					_ = c.String()
				}
			}
		}(g)
	}
	wg.Wait()

	if len(w.recs) == 0 {
		t.Fatal("nothing logged")
	}
	r := matchingo.NewOrderBook()
	if err := r.Replay(w.recs); err != nil {
		t.Fatal(err)
	}
	if got, want := r.Dump(), c.Dump(); !reflect.DeepEqual(got, want) {
		t.Fatalf("concurrent result is not equal to its serialized replay\n got %+v\nwant %+v", got, want)
	}
	if c.Sequence() == 0 {
		t.Fatal("no orders processed")
	}
}

func TestConcurrent_GetOrderReturnsStableCopy(t *testing.T) {
	c := matchingo.NewConcurrentOrderBook()
	c.Process(limit("a", matchingo.Buy, 5, 10))
	snap := c.GetOrder("a")
	c.Process(matchingo.NewMarketOrder("m", matchingo.Sell, d("2")))
	if !snap.Quantity().Equal(d("5")) {
		t.Fatalf("copy changed under matching: %s", snap.Quantity())
	}
	if live := c.GetOrder("a"); !live.Quantity().Equal(d("3")) {
		t.Fatalf("live quantity = %s, want 3", live.Quantity())
	}
	if c.GetOrder("nope") != nil {
		t.Fatal("unknown ID should return nil")
	}
}

func TestConcurrent_ViewAndUpdate(t *testing.T) {
	c := matchingo.NewConcurrentOrderBook()
	c.Update(func(ob *matchingo.OrderBook) { ob.Process(limit("a", matchingo.Buy, 1, 10)) })
	var n int
	c.View(func(ob *matchingo.OrderBook) { n = len(ob.Dump().Bids) })
	if n != 1 {
		t.Fatalf("bids = %d, want 1", n)
	}
}

func BenchmarkConcurrentBookProcess(b *testing.B) {
	c := matchingo.NewConcurrentOrderBook()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.Process(matchingo.NewLimitOrder(fmt.Sprint(i), matchingo.Buy, d("1"), d("10"), matchingo.GTC, ""))
		c.CancelOrder(fmt.Sprint(i))
	}
}
