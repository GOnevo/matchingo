package matchingo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// WALOp identifies the command a WALRecord stores.
type WALOp string

// Logged commands
const (
	WALProcess    WALOp = "process"
	WALReplace    WALOp = "replace"
	WALCancel     WALOp = "cancel"
	WALModify     WALOp = "modify"
	WALModifyStop WALOp = "modify_stop"
	WALExpire     WALOp = "expire"
	WALHalt       WALOp = "halt"
	WALResume     WALOp = "resume"
)

// WALRecord is one logged command. Time is the book clock reading when the command ran;
// replay feeds it back as the clock so GTD, TTL and expiry behave identically.
type WALRecord struct {
	Seq       uint64
	Time      time.Time
	Op        WALOp
	ID        string       // order ID: cancel, modify, modify_stop; the replaced order for replace
	Order     *OrderRecord // process, replace: the order as submitted
	Price     Decimal      // modify: new price; modify_stop: new limit price
	StopPrice Decimal      // modify_stop: new stop price
	Quantity  Decimal      // modify, modify_stop: new quantity
}

// WAL receives every command before it is applied to the book. Append must not return
// until the record is durable if crash safety is required. If Append fails the command
// is not applied, the book latches the error (see OrderBook.WALError) and rejects further
// mutating calls until SetWAL is called.
type WAL interface {
	Append(rec WALRecord) error
}

// WithWAL attaches a write-ahead log at construction time.
func WithWAL(w WAL) Option {
	return func(ob *OrderBook) { ob.wal = w }
}

// SetWAL attaches (or, with nil, detaches) the write-ahead log and clears a latched WAL error.
func (ob *OrderBook) SetWAL(w WAL) {
	ob.wal = w
	ob.walErr = nil
}

// WALError returns the latched error from a failed WAL append, or nil.
func (ob *OrderBook) WALError() error { return ob.walErr }

// WALSeq returns the sequence of the last record logged or replayed.
func (ob *OrderBook) WALSeq() uint64 { return ob.walSeq }

// appendWAL stamps and writes rec, then freezes the book clock at rec.Time until the returned
// restore func runs, so the command sees the same time replay will feed it.
func (ob *OrderBook) appendWAL(rec WALRecord) (restore func(), err error) {
	if ob.walErr != nil {
		return nil, ob.walErr
	}
	rec.Seq = ob.walSeq + 1
	if rec.Time.IsZero() {
		rec.Time = ob.clock()
	}
	if err := ob.wal.Append(rec); err != nil {
		ob.walErr = fmt.Errorf("%w: %w", ErrWAL, err)
		return nil, ob.walErr
	}
	ob.walSeq = rec.Seq
	orig := ob.clock
	ob.clock = func() time.Time { return rec.Time }
	return func() { ob.clock = orig }, nil
}

func (ob *OrderBook) processLogged(order *Order) (*Done, error) {
	if err := ob.validate(order); err != nil { // rejected commands are not logged
		return nil, err
	}
	if ob.GetOrder(order.ID()) != nil {
		return nil, ErrOrderExists
	}
	rec := orderToRecord(order)
	restore, err := ob.appendWAL(WALRecord{Op: WALProcess, Order: &rec})
	if err != nil {
		return nil, err
	}
	defer restore()
	return ob.process(order)
}

func (ob *OrderBook) replaceLogged(oldID string, newOrder *Order) (*Done, error) {
	if ob.GetOrder(oldID) == nil {
		return nil, ErrOrderNotFound
	}
	if err := ob.validate(newOrder); err != nil {
		return nil, err
	}
	if newOrder.ID() != oldID && ob.GetOrder(newOrder.ID()) != nil {
		return nil, ErrOrderExists
	}
	rec := orderToRecord(newOrder)
	restore, err := ob.appendWAL(WALRecord{Op: WALReplace, ID: oldID, Order: &rec})
	if err != nil {
		return nil, err
	}
	defer restore()
	return ob.replace(oldID, newOrder)
}

func (ob *OrderBook) cancelOrderLogged(orderID string) *Order {
	if ob.GetOrder(orderID) == nil {
		return nil
	}
	restore, err := ob.appendWAL(WALRecord{Op: WALCancel, ID: orderID})
	if err != nil {
		return nil
	}
	defer restore()
	return ob.cancelOrder(orderID)
}

func (ob *OrderBook) modifyLogged(orderID string, newPrice, newQty Decimal) (*Done, error) {
	restore, err := ob.appendWAL(WALRecord{Op: WALModify, ID: orderID, Price: newPrice, Quantity: newQty})
	if err != nil {
		return nil, err
	}
	defer restore()
	return ob.modify(orderID, newPrice, newQty)
}

func (ob *OrderBook) modifyStopLogged(orderID string, newStopPrice, newLimitPrice, newQty Decimal) error {
	rec := WALRecord{Op: WALModifyStop, ID: orderID, StopPrice: newStopPrice, Price: newLimitPrice, Quantity: newQty}
	restore, err := ob.appendWAL(rec)
	if err != nil {
		return err
	}
	defer restore()
	return ob.modifyStop(orderID, newStopPrice, newLimitPrice, newQty)
}

func (ob *OrderBook) haltLogged(op WALOp, halted bool) {
	restore, err := ob.appendWAL(WALRecord{Op: op})
	if err != nil {
		return
	}
	defer restore()
	ob.setHalted(halted, false)
}

func (ob *OrderBook) expireOrdersLogged(now time.Time) []*Order {
	restore, err := ob.appendWAL(WALRecord{Op: WALExpire, Time: now})
	if err != nil {
		return nil
	}
	defer restore()
	return ob.expireOrders(now)
}

// Replay re-applies records on top of the current state, normally a Restore of the latest
// snapshot. Records at or before WALSeq are skipped; the rest must follow without gaps.
// Each record runs with the book clock set to its logged time. The book's configuration
// (precision, instrument, circuit breaker) must match the one that wrote the log. Commands
// that were rejected when first run are rejected again, so their errors are ignored. The
// attached WAL, if any, is not written to during replay.
func (ob *OrderBook) Replay(records []WALRecord) error {
	wal, clock := ob.wal, ob.clock
	ob.wal = nil
	defer func() { ob.wal, ob.clock = wal, clock }()

	var now time.Time
	ob.clock = func() time.Time { return now }

	for _, rec := range records {
		if rec.Seq <= ob.walSeq {
			continue
		}
		if rec.Seq != ob.walSeq+1 {
			return fmt.Errorf("%w: expected %d, got %d", ErrWALGap, ob.walSeq+1, rec.Seq)
		}
		now = rec.Time
		switch rec.Op {
		case WALProcess, WALReplace:
			if rec.Order == nil {
				return fmt.Errorf("%w: %s record %d has no order", ErrInvalidWAL, rec.Op, rec.Seq)
			}
			if rec.Op == WALProcess {
				ob.Process(recordToOrder(*rec.Order))
			} else {
				ob.Replace(rec.ID, recordToOrder(*rec.Order))
			}
		case WALCancel:
			ob.CancelOrder(rec.ID)
		case WALModify:
			ob.Modify(rec.ID, rec.Price, rec.Quantity)
		case WALModifyStop:
			ob.ModifyStop(rec.ID, rec.StopPrice, rec.Price, rec.Quantity)
		case WALExpire:
			ob.ExpireOrders(rec.Time)
		case WALHalt:
			ob.Halt()
		case WALResume:
			ob.Resume()
		default:
			return fmt.Errorf("%w: unknown op %q at %d", ErrInvalidWAL, rec.Op, rec.Seq)
		}
		ob.walSeq = rec.Seq
	}
	return nil
}

type syncer interface{ Sync() error }

// JSONWAL writes records as one JSON object per line. If the writer has a Sync method
// (such as *os.File) it is called after every record, so a returned Append is durable.
type JSONWAL struct {
	enc *json.Encoder
	s   syncer
}

// NewJSONWAL returns a WAL that appends JSON lines to w.
func NewJSONWAL(w io.Writer) *JSONWAL {
	j := &JSONWAL{enc: json.NewEncoder(w)}
	j.s, _ = w.(syncer)
	return j
}

// Append implements WAL.
func (j *JSONWAL) Append(rec WALRecord) error {
	if err := j.enc.Encode(rec); err != nil {
		return err
	}
	if j.s != nil {
		return j.s.Sync()
	}
	return nil
}

// ReadWAL reads JSON-line records. A torn final record (a crash during Append, so the
// command was never applied) is ignored; any other malformed record is an error.
func ReadWAL(r io.Reader) ([]WALRecord, error) {
	dec := json.NewDecoder(r)
	var out []WALRecord
	for {
		var rec WALRecord
		err := dec.Decode(&rec)
		if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
			return out, nil
		}
		if err != nil {
			return out, fmt.Errorf("%w: %v", ErrInvalidWAL, err)
		}
		out = append(out, rec)
	}
}
