package orderbook

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
)

type walEnv struct {
	book *matchingo.OrderBook
	now  *time.Time
	buf  *bytes.Buffer
}

func newWALEnv() walEnv {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	buf := &bytes.Buffer{}
	b := matchingo.NewOrderBook(
		matchingo.WithClock(func() time.Time { return now }),
		matchingo.WithWAL(matchingo.NewJSONWAL(buf)),
	)
	return walEnv{b, &now, buf}
}

func freshBook(now *time.Time) *matchingo.OrderBook {
	return matchingo.NewOrderBook(matchingo.WithClock(func() time.Time { return *now }))
}

// workload exercises every logged command and order feature.
func workload(e walEnv, from, to int) {
	b, tick := e.book, func() { *e.now = e.now.Add(time.Second) }
	steps := []func(){
		func() { b.Process(limit("b1", matchingo.Buy, 5, 100)) },
		func() { b.Process(limit("a1", matchingo.Sell, 3, 105)) },
		func() {
			b.Process(matchingo.NewIcebergOrder("ice", matchingo.Sell, d("10"), d("2"), d("106"), matchingo.GTC, ""))
		},
		func() {
			b.Process(matchingo.NewGTDOrder("gtd", matchingo.Buy, d("1"), d("99"), e.now.Add(5*time.Second), ""))
		},
		func() {
			o := matchingo.NewLimitOrder("ttl", matchingo.Buy, d("1"), d("98"), matchingo.GTC, "")
			o.SetTTL(20 * time.Second)
			b.Process(o)
		},
		func() {
			b.Process(matchingo.NewTrailingStopOrder("tr1", matchingo.Sell, d("1"), d("90"), d("5"), ""))
			b.Process(matchingo.NewTrailingStopOrder("tr2", matchingo.Sell, d("1"), d("91"), d("6"), ""))
		},
		func() { b.Process(matchingo.NewStopLimitOrder("sl", matchingo.Buy, d("1"), d("110"), d("107"), "")) },
		func() { b.Process(matchingo.NewStopMarketOrder("sm", matchingo.Buy, d("1"), d("108"), "")) },
		func() {
			o := matchingo.NewLimitOrder("stp", matchingo.Buy, d("1"), d("97"), matchingo.GTC, "")
			o.SetSTP("p1", matchingo.STPCancelNewest)
			b.Process(o)
		},
		func() { b.Process(matchingo.NewMarketOrder("m1", matchingo.Buy, d("4"))) }, // sweeps asks, trails stops
		func() { b.Process(matchingo.NewMarketToLimitOrder("mtl", matchingo.Buy, d("50"))) },
		func() { b.Process(limit("b2", matchingo.Buy, 2, 100)) },
		func() { b.Modify("b2", d("100"), d("1")) },
		func() { b.ModifyStop("sl", d("107"), d("111"), d("2")) },
		func() { b.Replace("b1", limit("b1x", matchingo.Buy, 4, 101)) },
		func() { b.Halt() },
		func() { b.Process(limit("rejected", matchingo.Buy, 1, 1)) },
		func() { b.Resume() },
		func() { b.ExpireOrders(*e.now) },
		func() { b.CancelOrder("stp") },
		func() { b.Process(limit("last", matchingo.Sell, 1, 101)) },
	}
	for i := from; i < to && i < len(steps); i++ {
		steps[i]()
		tick()
	}
}

func recordsOf(t *testing.T, e walEnv) []matchingo.WALRecord {
	t.Helper()
	recs, err := matchingo.ReadWAL(bytes.NewReader(e.buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func TestWAL_ReplayFromEmptyReproducesState(t *testing.T) {
	e := newWALEnv()
	workload(e, 0, 100)
	recs := recordsOf(t, e)
	if len(recs) < 15 {
		t.Fatalf("only %d records logged", len(recs))
	}
	for _, r := range recs {
		if r.Order != nil && r.Order.ID == "rejected" {
			t.Fatal("rejected commands must not be logged")
		}
	}

	if s := e.book.Dump(); len(s.Bids) == 0 || len(s.Stops) < 2 || s.TradeID == 0 {
		t.Fatalf("workload too trivial: %d bids %d asks %d stops %d trades", len(s.Bids), len(s.Asks), len(s.Stops), s.TradeID)
	}

	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	r := freshBook(&start)
	if err := r.Replay(recs); err != nil {
		t.Fatal(err)
	}
	if got, want := r.Dump(), e.book.Dump(); !reflect.DeepEqual(got, want) {
		t.Fatalf("replayed state differs\n got %+v\nwant %+v", got, want)
	}
	if r.WALSeq() != e.book.WALSeq() {
		t.Fatalf("WALSeq %d != %d", r.WALSeq(), e.book.WALSeq())
	}
}

func TestWAL_SnapshotPlusTail(t *testing.T) {
	e := newWALEnv()
	workload(e, 0, 10)
	snap := e.book.Dump()
	if snap.WALSeq == 0 {
		t.Fatal("snapshot must record the WAL position")
	}
	workload(e, 10, 100)

	now := *e.now
	r := freshBook(&now)
	if err := r.Restore(snap); err != nil {
		t.Fatal(err)
	}
	if err := r.Replay(recordsOf(t, e)); err != nil { // full log: covered records are skipped
		t.Fatal(err)
	}
	if got, want := r.Dump(), e.book.Dump(); !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot+tail differs\n got %+v\nwant %+v", got, want)
	}
}

func TestWAL_GapAndUnknownOp(t *testing.T) {
	e := newWALEnv()
	workload(e, 0, 5)
	recs := recordsOf(t, e)
	now := *e.now
	if err := freshBook(&now).Replay(recs[1:]); !errors.Is(err, matchingo.ErrWALGap) {
		t.Fatalf("err = %v, want ErrWALGap", err)
	}
	bad := []matchingo.WALRecord{{Seq: 1, Op: "bogus"}}
	if err := freshBook(&now).Replay(bad); !errors.Is(err, matchingo.ErrInvalidWAL) {
		t.Fatalf("err = %v, want ErrInvalidWAL", err)
	}
}

func TestWAL_TornTailIgnored(t *testing.T) {
	e := newWALEnv()
	workload(e, 0, 3)
	n := len(recordsOf(t, e))
	torn := e.buf.Bytes()[:e.buf.Len()-10]
	recs, err := matchingo.ReadWAL(bytes.NewReader(torn))
	if err != nil || len(recs) != n-1 {
		t.Fatalf("records=%d err=%v, want %d records", len(recs), err, n-1)
	}
	if _, err := matchingo.ReadWAL(bytes.NewReader([]byte("{nope}\n"))); !errors.Is(err, matchingo.ErrInvalidWAL) {
		t.Fatalf("garbage: err = %v", err)
	}
}

type failingWAL struct{ fail bool }

func (f *failingWAL) Append(matchingo.WALRecord) error {
	if f.fail {
		return errors.New("disk full")
	}
	return nil
}

func TestWAL_FailureBlocksCommandAndLatches(t *testing.T) {
	w := &failingWAL{}
	b := matchingo.NewOrderBook(matchingo.WithWAL(w))
	b.Process(limit("a", matchingo.Buy, 1, 10))

	w.fail = true
	if _, err := b.Process(limit("b", matchingo.Buy, 1, 10)); !errors.Is(err, matchingo.ErrWAL) {
		t.Fatalf("err = %v, want ErrWAL", err)
	}
	if b.GetOrder("b") != nil || b.Sequence() != 1 {
		t.Fatal("command must not be applied when the WAL write fails")
	}
	w.fail = false
	if _, err := b.Process(limit("c", matchingo.Buy, 1, 10)); !errors.Is(err, matchingo.ErrWAL) || b.WALError() == nil {
		t.Fatal("error must stay latched until SetWAL")
	}
	if b.CancelOrder("a") != nil || b.GetOrder("a") == nil {
		t.Fatal("cancel must also be blocked while latched")
	}
	b.SetWAL(w)
	if _, err := b.Process(limit("c", matchingo.Buy, 1, 10)); err != nil {
		t.Fatalf("after SetWAL: %v", err)
	}
}

// With the default (real) clock, the logged time and the order's own timestamps must be one reading.
func TestWAL_RealClockReplayIsExact(t *testing.T) {
	buf := &bytes.Buffer{}
	b := matchingo.NewOrderBook(matchingo.WithWAL(matchingo.NewJSONWAL(buf)))
	o := matchingo.NewLimitOrder("ttl", matchingo.Buy, d("1"), d("10"), matchingo.GTC, "")
	o.SetTTL(time.Hour)
	b.Process(o)
	b.Process(limit("plain", matchingo.Buy, 1, 9))

	recs, err := matchingo.ReadWAL(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	r := matchingo.NewOrderBook()
	if err := r.Replay(recs); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Dump(), b.Dump()) {
		t.Fatalf("real-clock replay differs\n got %+v\nwant %+v", r.Dump(), b.Dump())
	}
}
