package matchingo

import (
	"errors"
	"time"
)

const snapshotVersion = 1

// OrderRecord is a plain-data copy of an Order for snapshot purposes.
type OrderRecord struct {
	ID          string
	OrderType   OrderType
	Side        Side
	IsQuote     bool
	Quantity    Decimal
	OriginalQty Decimal
	Price       Decimal
	Stop        Decimal
	TIF         TIF
	OCO         string
	PeakQty     Decimal
	HiddenQty   Decimal
	PostOnly    bool
	CreatedAt   time.Time
}

// Snapshot is a serializable snapshot of the full engine state.
type Snapshot struct {
	Version  int
	Sequence uint64
	TradeID  uint64
	Bids     []OrderRecord
	Asks     []OrderRecord
	Stops    []OrderRecord
	OCO      []string
}

// Dump returns a plain-data snapshot of the current engine state.
func (ob *OrderBook) Dump() Snapshot {
	snap := Snapshot{
		Version:  snapshotVersion,
		Sequence: ob.sequence,
		TradeID:  ob.tradeID,
	}

	for _, price := range ob.bids.Prices() {
		q := ob.bids.prices[price]
		for i := 0; i < q.Orders.Len(); i++ {
			snap.Bids = append(snap.Bids, orderToRecord(q.Orders.At(i)))
		}
	}

	for _, price := range ob.asks.Prices() {
		q := ob.asks.prices[price]
		for i := 0; i < q.Orders.Len(); i++ {
			snap.Asks = append(snap.Asks, orderToRecord(q.Orders.At(i)))
		}
	}

	for id := range ob.Stop.orders {
		snap.Stops = append(snap.Stops, orderToRecord(ob.Stop.orders[id]))
	}

	for id := range ob.OCO {
		snap.OCO = append(snap.OCO, id)
	}

	return snap
}

// Restore clears the engine state and loads from snap.
// Returns an error if the snapshot version is not supported.
func (ob *OrderBook) Restore(snap Snapshot) error {
	if snap.Version != snapshotVersion {
		return errors.New("unsupported snapshot version")
	}

	// Clear current state.
	ob.orders = map[string]*Order{}
	ob.bids = NewOrderSideBid()
	ob.asks = NewOrderSideAsk()
	ob.Stop = NewStopBook()
	ob.OCO = map[string]struct{}{}
	ob.sequence = snap.Sequence
	ob.tradeID = snap.TradeID

	for _, rec := range snap.Bids {
		o := recordToOrder(rec)
		ob.bids.Append(o)
		ob.orders[o.id] = o
	}

	for _, rec := range snap.Asks {
		o := recordToOrder(rec)
		ob.asks.Append(o)
		ob.orders[o.id] = o
	}

	for _, rec := range snap.Stops {
		o := recordToOrder(rec)
		ob.Stop.Append(o)
		ob.orders[o.id] = o
	}

	for _, id := range snap.OCO {
		ob.OCO[id] = struct{}{}
	}

	return nil
}

func orderToRecord(o *Order) OrderRecord {
	return OrderRecord{
		ID:          o.id,
		OrderType:   o.orderType,
		Side:        o.side,
		IsQuote:     o.isQuote,
		Quantity:    o.quantity,
		OriginalQty: o.originalQty,
		Price:       o.price,
		Stop:        o.stop,
		TIF:         o.tif,
		OCO:         o.oco,
		PeakQty:     o.peakQty,
		HiddenQty:   o.hiddenQty,
		PostOnly:    o.postOnly,
		CreatedAt:   o.createdAt,
	}
}

func recordToOrder(rec OrderRecord) *Order {
	return &Order{
		id:          rec.ID,
		orderType:   rec.OrderType,
		side:        rec.Side,
		isQuote:     rec.IsQuote,
		quantity:    rec.Quantity,
		originalQty: rec.OriginalQty,
		price:       rec.Price,
		stop:        rec.Stop,
		tif:         rec.TIF,
		oco:         rec.OCO,
		peakQty:     rec.PeakQty,
		hiddenQty:   rec.HiddenQty,
		postOnly:    rec.PostOnly,
		createdAt:   rec.CreatedAt,
	}
}
