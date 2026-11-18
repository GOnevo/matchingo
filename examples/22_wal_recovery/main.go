// Example 22 — Write-Ahead Log and Crash Recovery
//
// Every command (Process, Replace, Cancel, Modify, ModifyStop, ExpireOrders,
// Halt, Resume) is written to the WAL *before* it is applied. After a crash,
// restore the latest snapshot and replay the log: records already covered by
// the snapshot (Snapshot.WALSeq) are skipped, the rest are re-applied with the
// logged clock, so the book comes back exactly as it was.
//
// Concepts shown:
//   - matchingo.WithWAL(matchingo.NewJSONWAL(file)) — JSON lines, fsync per record on *os.File
//   - ob.Dump() — snapshots carry WALSeq
//   - matchingo.ReadWAL + ob.Restore + ob.Replay — recovery
//   - A failed Append blocks the command and latches ob.WALError()
package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/gonevo/matchingo"
)

func main() {
	f, _ := os.CreateTemp("", "matchingo-*.wal")
	defer os.Remove(f.Name())

	ob := matchingo.NewOrderBook(matchingo.WithWAL(matchingo.NewJSONWAL(f)))
	ob.Process(matchingo.NewLimitOrder("bid1", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask1", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(105), matchingo.GTC, ""))

	snap := ob.Dump() // periodic snapshot; WALSeq marks how much of the log it covers
	fmt.Println("snapshot covers WAL records up to", snap.WALSeq)

	// More activity after the snapshot...
	ob.Process(matchingo.NewMarketOrder("m1", matchingo.Buy, matchingo.FromInt(2)))
	ob.CancelOrder("bid1")
	ob.Process(matchingo.NewLimitOrder("bid2", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(99), matchingo.GTC, ""))

	// ...then the process dies. Recover from the snapshot plus the log.
	f.Seek(0, 0)
	records, err := matchingo.ReadWAL(f)
	if err != nil {
		panic(err)
	}
	recovered := matchingo.NewOrderBook()
	if err := recovered.Restore(snap); err != nil {
		panic(err)
	}
	if err := recovered.Replay(records); err != nil {
		panic(err)
	}
	fmt.Printf("log has %d records, replayed %d after the snapshot\n", len(records), len(records)-int(snap.WALSeq))
	fmt.Println("recovered book identical to the live one:", reflect.DeepEqual(recovered.Dump(), ob.Dump()))
}
