package matchingo

// EventType identifies the kind of event emitted by the order book.
type EventType string

const (
	EventVolume    EventType = "volume"
	EventDepth     EventType = "depth"
	EventTrade     EventType = "trade"
	EventCanceled  EventType = "canceled"
	EventActivated EventType = "activated"
)

// VolumeEventData carries the data for a volume event.
type VolumeEventData struct {
	Side   Side
	Price  Decimal
	Volume Decimal
}

// TradeEventData carries the data for a trade event.
type TradeEventData struct {
	OrderID string
	Price   Decimal
	Amount  Decimal
	Role    string // "maker" or "taker"
}

// Event is a single order book event sent over a channel emitter.
type Event struct {
	Type EventType
	Data any
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

// Emit sends the event to the channel, dropping or blocking based on dropOnFull.
func (c *ChannelEmitter) Emit(eventType EventType, data any) {
	e := Event{Type: eventType, Data: data}
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
	if ob.emitter != nil {
		ob.emitter.Emit(eventType, data)
	}
}
