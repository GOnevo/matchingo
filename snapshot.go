package matchingo

import (
	"errors"
	"sort"
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
	Trail       Decimal
	TrailPct    bool
	ExpiresAt   time.Time
	Participant string
	STP         STPMode
	TTL         time.Duration // pending lifetime of a not-yet-processed order (WAL records)
	ToLimit     bool          // pending market-to-limit flag (WAL records)
	CreatedAt   time.Time
}

// Snapshot is a serializable snapshot of the full engine state.
type Snapshot struct {
	Version   int
	Sequence  uint64
	TradeID   uint64
	WALSeq    uint64 // last WAL record covered by this snapshot
	Halted    bool
	LastPrice Decimal
	Bids      []OrderRecord
	Asks      []OrderRecord
	Stops     []OrderRecord
	OCO       []string
}

// Dump returns a plain-data snapshot of the current engine state.
func (ob *OrderBook) Dump() Snapshot {
	snap := Snapshot{
		Version:   snapshotVersion,
		Sequence:  ob.sequence,
		TradeID:   ob.tradeID,
		WALSeq:    ob.walSeq,
		Halted:    ob.halted,
		LastPrice: ob.lastPrice,
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

	for _, o := range ob.Stop.inOrder() {
		snap.Stops = append(snap.Stops, orderToRecord(o))
	}

	for id := range ob.OCO {
		snap.OCO = append(snap.OCO, id)
	}
	sort.Strings(snap.OCO)

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
	ob.walSeq = snap.WALSeq
	ob.halted = snap.Halted
	ob.lastPrice = snap.LastPrice

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

	ob.refreshNextExpiry()

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
		Trail:       o.extra().trail,
		TrailPct:    o.extra().trailPct,
		ExpiresAt:   o.extra().expiresAt,
		Participant: o.extra().participant,
		STP:         o.extra().stp,
		TTL:         o.extra().ttl,
		ToLimit:     o.extra().toLimit,
		CreatedAt:   o.createdAt,
	}
}

func recordToOrder(rec OrderRecord) *Order {
	o := &Order{
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
	if rec.Trail.GreaterThan(Zero) || !rec.ExpiresAt.IsZero() || rec.Participant != "" || rec.STP != STPNone || rec.TTL > 0 || rec.ToLimit {
		o.ext = &orderExt{
			trail:       rec.Trail,
			trailPct:    rec.TrailPct,
			expiresAt:   rec.ExpiresAt,
			participant: rec.Participant,
			stp:         rec.STP,
			ttl:         rec.TTL,
			toLimit:     rec.ToLimit,
		}
	}
	return o
}
