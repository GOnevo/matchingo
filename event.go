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

// EventEmitter receives events from the order book.
type EventEmitter interface {
	Emit(eventType EventType, data any)
}

// SetEmitter attaches an optional emitter to the order book.
// Pass nil to disable events.
func (ob *OrderBook) SetEmitter(e EventEmitter) {
	ob.emitter = e
}

func (ob *OrderBook) emit(eventType EventType, data any) {
	if ob.emitter != nil {
		ob.emitter.Emit(eventType, data)
	}
}
