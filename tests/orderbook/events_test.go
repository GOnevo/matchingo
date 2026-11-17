package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

// recordedEvent captures one emitted event.
type recordedEvent struct {
	eventType matchingo.EventType
	data      any
}

// testEmitter collects all emitted events.
type testEmitter struct {
	events []recordedEvent
}

func (e *testEmitter) Emit(eventType matchingo.EventType, data any) {
	e.events = append(e.events, recordedEvent{eventType: eventType, data: data})
}

func (e *testEmitter) eventsOf(t matchingo.EventType) []recordedEvent {
	var out []recordedEvent
	for _, ev := range e.events {
		if ev.eventType == t {
			out = append(out, ev)
		}
	}
	return out
}

func newObWithEmitter() (*matchingo.OrderBook, *testEmitter) {
	ob := matchingo.NewOrderBook()
	em := &testEmitter{}
	ob.SetEmitter(em)
	return ob, em
}

// §18.16: Volume event fires when order rests.
func TestEvent_VolumeOnRest(t *testing.T) {
	ob, em := newObWithEmitter()
	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))

	vols := em.eventsOf(matchingo.EventVolume)
	if len(vols) == 0 {
		t.Fatal("expected volume event when order rests, got none")
	}
	v := vols[0].data.(matchingo.VolumeEventData)
	if v.Side != matchingo.Buy {
		t.Errorf("expected Buy side, got %v", v.Side)
	}
	if !v.Price.Equal(matchingo.FromInt(100)) {
		t.Errorf("expected price 100, got %v", v.Price)
	}
	if !v.Volume.Equal(matchingo.FromInt(5)) {
		t.Errorf("expected volume 5, got %v", v.Volume)
	}
}

// §18.16: Volume event fires after trade reduces volume.
func TestEvent_VolumeAfterTrade(t *testing.T) {
	ob, em := newObWithEmitter()
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	em.events = nil // reset; we only care about the trade-triggered volume event

	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(100), "", ""))

	vols := em.eventsOf(matchingo.EventVolume)
	found := false
	for _, ev := range vols {
		v := ev.data.(matchingo.VolumeEventData)
		if v.Side == matchingo.Sell && v.Price.Equal(matchingo.FromInt(100)) && v.Volume.Equal(matchingo.FromInt(7)) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected volume event with sell/100/7 after partial trade; events: %v", vols)
	}
}

// §18.16: Volume event fires after cancel.
func TestEvent_VolumeAfterCancel(t *testing.T) {
	ob, em := newObWithEmitter()
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	em.events = nil

	ob.CancelOrder("sell-1")

	vols := em.eventsOf(matchingo.EventVolume)
	if len(vols) == 0 {
		t.Fatal("expected volume event after cancel, got none")
	}
	v := vols[0].data.(matchingo.VolumeEventData)
	if !v.Volume.Equal(matchingo.Zero) {
		t.Errorf("expected volume 0 after cancel, got %v", v.Volume)
	}
}

// §18.16: One canonical trade event per fill with both order IDs, correct price, and unique TradeID.
func TestEvent_TradeEvents(t *testing.T) {
	ob, em := newObWithEmitter()
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))
	em.events = nil

	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))

	trades := em.eventsOf(matchingo.EventTrade)
	if len(trades) != 1 {
		t.Fatalf("expected exactly 1 trade event per fill, got %d", len(trades))
	}

	td := trades[0].data.(matchingo.TradeEventData)
	if td.MakerID != "sell-1" {
		t.Errorf("expected MakerID=sell-1, got %s", td.MakerID)
	}
	if td.TakerID != "buy-1" {
		t.Errorf("expected TakerID=buy-1, got %s", td.TakerID)
	}
	if !td.Price.Equal(matchingo.FromInt(100)) {
		t.Errorf("expected trade Price=100, got %s", td.Price)
	}
	if td.Amount.Equal(matchingo.Zero) {
		t.Errorf("trade event amount should not be zero")
	}
	if td.TradeID == 0 {
		t.Errorf("expected non-zero TradeID")
	}
}

// §18.16: Canceled event fires when order is canceled.
func TestEvent_CanceledEvent(t *testing.T) {
	ob, em := newObWithEmitter()
	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))
	em.events = nil

	ob.CancelOrder("buy-1")

	canceled := em.eventsOf(matchingo.EventCanceled)
	if len(canceled) == 0 {
		t.Fatal("expected canceled event, got none")
	}
	id := canceled[0].data.(string)
	if id != "buy-1" {
		t.Errorf("expected order id buy-1, got %v", id)
	}
}

// §18.16: Activated event fires when stop order activates.
func TestEvent_ActivatedEvent(t *testing.T) {
	ob, em := newObWithEmitter()
	// Place a resting sell so the activated stop order can match.
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	ob.Process(matchingo.NewStopLimitOrder("stop-1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.FromInt(99), ""))
	em.events = nil

	// Trigger activation by processing a trade at price 99.
	ob.Process(matchingo.NewLimitOrder("sell-trigger", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(99), "", ""))
	ob.Process(matchingo.NewLimitOrder("buy-trigger", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(99), "", ""))

	activated := em.eventsOf(matchingo.EventActivated)
	if len(activated) == 0 {
		t.Fatal("expected activated event, got none")
	}
	id := activated[0].data.(string)
	if id != "stop-1" {
		t.Errorf("expected stop-1, got %v", id)
	}
}

// §18.16: Depth event fires when depth is queried.
func TestEvent_DepthEvent(t *testing.T) {
	ob, em := newObWithEmitter()
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))

	ob.Depth()

	depths := em.eventsOf(matchingo.EventDepth)
	if len(depths) == 0 {
		t.Fatal("expected depth event when Depth() is called, got none")
	}
}

// §18.16: BBO event fires when best bid or ask changes.
func TestEvent_BBOEvent(t *testing.T) {
	ob, em := newObWithEmitter()

	// Place a resting sell — BBO ask changes from nil to 100.
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))
	bbos := em.eventsOf(matchingo.EventBBO)
	if len(bbos) == 0 {
		t.Fatal("expected BBO event after resting sell, got none")
	}
	d := bbos[len(bbos)-1].data.(matchingo.BBOEventData)
	if d.Ask == nil || !d.Ask.Price.Equal(matchingo.FromInt(100)) {
		t.Errorf("expected ask=100, got %v", d.Ask)
	}

	em.events = nil

	// Cancel the sell — BBO ask changes back to nil.
	ob.CancelOrder("sell-1")
	bbos = em.eventsOf(matchingo.EventBBO)
	if len(bbos) == 0 {
		t.Fatal("expected BBO event after cancel, got none")
	}
	d = bbos[len(bbos)-1].data.(matchingo.BBOEventData)
	if d.Ask != nil {
		t.Errorf("expected ask=nil after cancel, got %v", d.Ask)
	}

	em.events = nil

	// Place sell and buy that trade fully — BBO fires when book changes.
	ob.Process(matchingo.NewLimitOrder("sell-2", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(50), "", ""))
	em.events = nil
	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(50), "", ""))
	bbos = em.eventsOf(matchingo.EventBBO)
	if len(bbos) == 0 {
		t.Fatal("expected BBO event after trade clears level, got none")
	}
}

// §18.16: BBO event does NOT fire when BBO is unchanged (e.g. mid-book cancel).
func TestEvent_NoBBOWhenUnchanged(t *testing.T) {
	ob, em := newObWithEmitter()
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))
	ob.Process(matchingo.NewLimitOrder("sell-2", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(110), "", ""))
	em.events = nil

	// Cancel sell-2 (110) — BBO ask is still sell-1 at 100, no change.
	ob.CancelOrder("sell-2")
	bbos := em.eventsOf(matchingo.EventBBO)
	if len(bbos) != 0 {
		t.Errorf("expected no BBO event when BBO unchanged, got %d", len(bbos))
	}
}

// §18.16: No events when emitter is disabled (nil).
func TestEvent_NoEventsWhenDisabled(t *testing.T) {
	ob := matchingo.NewOrderBook()
	// No emitter set — all operations must complete without panic.
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))
	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))
	ob.Process(matchingo.NewLimitOrder("sell-2", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(110), "", ""))
	ob.CancelOrder("sell-2")
	ob.Depth()
	// If we reach here without panic, the nil-emitter guard works.
}
