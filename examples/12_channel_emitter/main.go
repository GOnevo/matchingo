// Example 12 — Channel Emitter (async events)
//
// SetEventChannel attaches a built-in ChannelEmitter that sends every engine
// event to a chan<- Event you own. Your consumer goroutine reads at its own pace.
//
// dropOnFull=true  — Process() never blocks; events are dropped when the channel
//
//	is full (use a large buffer for low-loss fire-and-forget).
//
// dropOnFull=false — Process() blocks until the channel has space (backpressure).
//
// Concepts shown:
//   - matchingo.WithEmitter(matchingo.NewChannelEmitter(ch, dropOnFull)) constructor option
//   - ob.SetEventChannel(ch, dropOnFull) for post-construction attachment
//   - Reading matchingo.Event{Type, Data, Sequence} in a goroutine
//   - Closing the channel after the book is done
package main

import (
	"fmt"
	"sync"

	"github.com/gonevo/matchingo"
)

func main() {
	ch := make(chan matchingo.Event, 64)
	ob := matchingo.NewOrderBook(
		matchingo.WithEmitter(matchingo.NewChannelEmitter(ch, true)), // drop on full — Process() stays non-blocking
	)

	var wg sync.WaitGroup
	wg.Go(func() {
		for e := range ch {
			switch e.Type {
			case matchingo.EventTrade:
				d := e.Data.(matchingo.TradeEventData)
				fmt.Printf("  [TRADE]     id=%d  maker=%s  taker=%s  price=%s  amount=%s\n",
					d.TradeID, d.MakerID, d.TakerID, d.Price, d.Amount)
			case matchingo.EventVolume:
				d := e.Data.(matchingo.VolumeEventData)
				fmt.Printf("  [VOLUME]    side=%s  price=%s  volume=%s\n",
					d.Side, d.Price, d.Volume)
			case matchingo.EventCanceled:
				fmt.Printf("  [CANCELED]  id=%s\n", e.Data)
			case matchingo.EventActivated:
				fmt.Printf("  [ACTIVATED] id=%s\n", e.Data)
			default:
				fmt.Printf("  [%s] %v\n", e.Type, e.Data)
			}
		}
	})

	fmt.Println("=== placing resting sell ===")
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))

	fmt.Println("\n=== matching buy ===")
	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(100), matchingo.GTC, ""))

	fmt.Println("\n=== canceling leftover sell ===")
	ob.CancelOrder("sell-1")

	// Detach emitter and close the channel to signal the consumer to stop.
	ob.SetEmitter(nil)
	close(ch)
	wg.Wait()
}
