// Example 13 — Sequence Numbers
//
// Every successful Process call increments a monotonic uint64 counter.
// The counter is stamped on:
//   - Done.Sequence       — the return value of Process
//   - TradeEventData.Sequence — the book sequence stamped on each EventTrade
//   - Event.Sequence      — on every event when using ChannelEmitter
//   - Snapshot.Sequence   — preserved across Dump / Restore
//
// This enables gap detection, ordered replay, and audit logs without
// relying on an external clock.
package main

import (
	"fmt"
	"sync"

	"github.com/gonevo/matchingo"
)

func main() {
	ob := matchingo.NewOrderBook()
	ch := make(chan matchingo.Event, 32)
	ob.SetEventChannel(ch, true)

	var wg sync.WaitGroup
	wg.Go(func() {
		for e := range ch {
			if e.Type == matchingo.EventTrade {
				d := e.Data.(matchingo.TradeEventData)
				fmt.Printf("  event seq=%-3d  [TRADE] id=%-4d  maker=%-8s taker=%-8s amount=%s\n",
					e.Sequence, d.TradeID, d.MakerID, d.TakerID, d.Amount)
			}
		}
	})

	fmt.Println("=== processing orders ===")

	d1, _ := ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), matchingo.GTC, ""))
	fmt.Printf("after sell-1  ob.Sequence=%d  done.Sequence=%d\n", ob.Sequence(), d1.Sequence)

	d2, _ := ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(100), matchingo.GTC, ""))
	fmt.Printf("after buy-1   ob.Sequence=%d  done.Sequence=%d\n", ob.Sequence(), d2.Sequence)

	ob.SetEmitter(nil)
	close(ch)
	wg.Wait()

	fmt.Println("\n=== snapshot round-trip ===")
	snap := ob.Dump()
	fmt.Printf("snapshot.Sequence=%d\n", snap.Sequence)

	ob2 := matchingo.NewOrderBook()
	ob2.Restore(snap)
	fmt.Printf("restored ob.Sequence=%d\n", ob2.Sequence())

	d3, _ := ob2.Process(matchingo.NewLimitOrder("sell-2", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(99), matchingo.GTC, ""))
	fmt.Printf("after sell-2  ob.Sequence=%d  done.Sequence=%d  (resumes from snapshot)\n", ob2.Sequence(), d3.Sequence)
}
