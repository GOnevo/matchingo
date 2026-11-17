// Example 09 — Snapshot (Dump & Restore)
//
// The order book can be serialized to a plain-data Snapshot and restored
// later.  This is useful for:
//   - Persistence (save to a database, reload on restart)
//   - Hot standby / replication (send snapshot to a replica)
//   - Testing (reproduce a specific book state deterministically)
//
// Dump captures: bids, asks, stop orders, and OCO pairs.
// Restore clears the engine and rebuilds from the snapshot.
//
// Concepts shown:
//   - ob.Dump() → Snapshot
//   - ob.Restore(snap)
//   - Snapshot is serialization-friendly (plain structs, no unexported fields)
package main

import (
	"encoding/json"
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()

	// Build a non-trivial book state.
	ob.Process(matchingo.NewLimitOrder("bid-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(99), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("bid-2", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(98), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder("ask-1", matchingo.Sell, matchingo.FromInt(8), matchingo.FromInt(101), matchingo.GTC, ""))
	ob.Process(matchingo.NewStopLimitOrder("stop-1", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(95), matchingo.FromInt(97), ""))
	fmt.Println("Original book:")
	fmt.Println(ob)

	// ------------------------------------------------------------------ //
	// Dump — returns a plain Snapshot struct, easy to marshal to JSON.   //
	// ------------------------------------------------------------------ //
	snap := ob.Dump()
	raw, _ := json.MarshalIndent(snap, "", "  ")
	fmt.Printf("\nSnapshot JSON (%d bytes):\n%s\n", len(raw), raw)

	// ------------------------------------------------------------------ //
	// Restore into a brand-new engine — state is identical.              //
	// ------------------------------------------------------------------ //
	ob2 := matchingo.NewOrderBook()
	if err := ob2.Restore(snap); err != nil {
		panic(err)
	}
	fmt.Println("\nRestored book:")
	fmt.Println(ob2)

	// Prove the restored book still works: add a crossing ask.
	done, _ := ob2.Process(matchingo.NewLimitOrder("ask-cross", matchingo.Sell, matchingo.FromInt(4), matchingo.FromInt(99), matchingo.GTC, ""))
	fmt.Printf("\nCross after restore → processed=%s  trades=%d\n",
		done.Processed, len(done.Trades)-1)
	fmt.Println(ob2)
}
