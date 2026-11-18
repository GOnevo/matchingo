// Example 17 — Good-Till-Date (GTD) Orders
//
// A GTD order is a limit order that the engine cancels once its expiry time
// has passed. The engine has no background timer: expired orders are removed
// when the next order is processed, or when you call ExpireOrders from your own
// timer. An order with a "Day" lifetime is a GTD order expiring at the close.
//
// Concepts shown:
//   - matchingo.NewGTDOrder(id, side, qty, price, expiresAt, oco)
//   - matchingo.WithClock — inject a clock (default time.Now), handy in tests
//   - ob.ExpireOrders(now) — explicit sweep; returns the canceled orders
//   - order.SetTTL(d) — relative lifetime from submission (works on stop orders too)
package main

import (
	"fmt"
	"time"

	"github.com/gonevo/matchingo"
)

func main() {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ob := matchingo.NewOrderBook(matchingo.WithClock(func() time.Time { return now }))

	ob.Process(matchingo.NewGTDOrder("day-bid", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), now.Add(8*time.Hour), ""))
	fmt.Println("09:00 resting:", ob.GetOrder("day-bid") != nil)

	now = now.Add(9 * time.Hour) // market close passes
	done, _ := ob.Process(matchingo.NewLimitOrder("other", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(99), matchingo.GTC, ""))
	fmt.Println("18:00 canceled by the engine:", done.Canceled, "resting:", ob.GetOrder("day-bid") != nil)

	// Idle book: sweep from a timer instead.
	ob.Process(matchingo.NewGTDOrder("g2", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(110), now.Add(time.Minute), ""))
	for _, o := range ob.ExpireOrders(now.Add(time.Hour)) {
		fmt.Println("ExpireOrders canceled:", o.ID())
	}

	// TTL: expires 30s after submission.
	ttl := matchingo.NewLimitOrder("ttl", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(95), matchingo.GTC, "")
	ttl.SetTTL(30 * time.Second)
	ob.Process(ttl)
	fmt.Println("ttl expires at:", ttl.ExpiresAt().Format(time.TimeOnly))
}
