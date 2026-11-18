package matchingo

// EventType identifies the kind of event emitted by the order book.
type EventType string

const (
	EventVolume    EventType = "volume"
	EventDepth     EventType = "depth"
	EventTrade     EventType = "trade"
	EventCanceled  EventType = "canceled"
	EventActivated EventType = "activated"
	EventBBO       EventType = "bbo"
	EventHalt      EventType = "halt"
)

// BBOEventData carries the best bid and best ask after a change.
// Either field is nil when that side of the book is empty.
type BBOEventData struct {
	Bid *Level
	Ask *Level
}

// HaltEventData carries the trading state after a halt or resume.
// Auto is true when the circuit breaker (not Halt) caused a halt.
type HaltEventData struct {
	Halted bool
	Auto   bool
}

// VolumeEventData carries the data for a volume event.
type VolumeEventData struct {
	Side   Side
	Price  Decimal
	Volume Decimal
}

// TradeEventData carries the data for a single canonical trade event.
// One event is emitted per fill; it names both sides and the actual trade price.
type TradeEventData struct {
	TradeID  uint64
	MakerID  string
	TakerID  string
	Price    Decimal
	Amount   Decimal
	Sequence uint64
}

// Event is a single order book event sent over a channel emitter.
type Event struct {
	Type     EventType
	Data     any
	Sequence uint64
}

// EventEmitter receives events from the order book.
type EventEmitter interface {
	Emit(eventType EventType, data any)
}

// ChannelEmitter implements EventEmitter by sending events to a channel.
// When dropOnFull is true, events are silently dropped if the channel is full.
// When false, Emit blocks until the channel has space (backpressure).
type ChannelEmitter struct {
	ch         chan<- Event
	dropOnFull bool
}

// NewChannelEmitter returns an EventEmitter backed by ch.
// The caller owns ch and reads from it in their own goroutine.
func NewChannelEmitter(ch chan<- Event, dropOnFull bool) *ChannelEmitter {
	return &ChannelEmitter{ch: ch, dropOnFull: dropOnFull}
}

// Emit implements EventEmitter. Sequence is always 0; use EmitSeq for a stamped event.
func (c *ChannelEmitter) Emit(eventType EventType, data any) {
	c.EmitSeq(0, eventType, data)
}

// EmitSeq sends a sequenced event to the channel.
func (c *ChannelEmitter) EmitSeq(seq uint64, eventType EventType, data any) {
	e := Event{Type: eventType, Data: data, Sequence: seq}
	if c.dropOnFull {
		select {
		case c.ch <- e:
		default:
		}
		return
	}
	c.ch <- e
}

// SetEmitter attaches an optional emitter to the order book.
// Pass nil to disable events.
func (ob *OrderBook) SetEmitter(e EventEmitter) {
	ob.emitter = e
}

// SetEventChannel is a convenience wrapper that creates a ChannelEmitter from ch
// and attaches it. dropOnFull controls behaviour when the channel is full.
func (ob *OrderBook) SetEventChannel(ch chan<- Event, dropOnFull bool) {
	ob.emitter = NewChannelEmitter(ch, dropOnFull)
}

func (ob *OrderBook) emit(eventType EventType, data any) {
	if ob.emitter == nil {
		return
	}
	if ce, ok := ob.emitter.(*ChannelEmitter); ok {
		ce.EmitSeq(ob.sequence, eventType, data)
		return
	}
	ob.emitter.Emit(eventType, data)
}
