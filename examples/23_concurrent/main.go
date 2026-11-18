// Example 23 — Concurrent Use
//
// OrderBook is single-threaded by design (no locks, zero overhead).
// ConcurrentOrderBook wraps one with a read/write lock so many goroutines can
// submit, cancel and read at the same time. Each call is atomic, and the outcome
// equals some single-threaded order of the calls.
//
// Concepts shown:
//   - matchingo.NewConcurrentOrderBook(opts...) — same options as NewOrderBook
//   - GetOrder returns a copy that does not change as the book matches
//   - View / Update for anything not wrapped
//   - Emitters and WALs run under the lock: they must not call back into the book
package main

import (
	"fmt"
	"sync"

	"github.com/gonevo/matchingo"
)

func main() {
	book := matchingo.NewConcurrentOrderBook()

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				id := fmt.Sprintf("g%d-%d", g, i)
				side := matchingo.Buy
				if g%2 == 1 {
					side = matchingo.Sell
				}
				book.Process(matchingo.NewLimitOrder(id, side, matchingo.FromInt(1), matchingo.FromInt(100), matchingo.GTC, ""))
				book.BestBid() // concurrent reads are fine
			}
		}(g)
	}
	wg.Wait()

	// 200 buys and 200 sells at the same price net out completely.
	fmt.Println("orders processed:", book.Sequence())
	fmt.Println("best bid:", book.BestBid(), "best ask:", book.BestAsk())

	// Anything not wrapped: run it under the lock.
	book.View(func(ob *matchingo.OrderBook) {
		fmt.Println("resting stop orders:", ob.Stop.Len())
	})
}
