// Example 20 — Trading Halts and Circuit Breaker
//
// While halted, the book rejects Process, Replace, Modify and ModifyStop with
// ErrTradingHalted. Cancels and expiry still work and the book stays intact.
//
// A halt can be manual (ob.Halt / ob.Resume) or automatic: a circuit breaker
// halts the book when a trade price moves more than a percentage away from the
// previous trade price. The order that caused the trip still completes.
//
// Concepts shown:
//   - ob.Halt(), ob.Resume(), ob.IsHalted()
//   - matchingo.WithCircuitBreaker(percent) / ob.SetCircuitBreaker
//   - EventHalt carries HaltEventData{Halted, Auto}
package main

import (
	"errors"
	"fmt"

	"github.com/gonevo/matchingo"
)

func trade(ob *matchingo.OrderBook, id string, price int) {
	ob.Process(matchingo.NewLimitOrder(id+"-ask", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(price), matchingo.GTC, ""))
	ob.Process(matchingo.NewLimitOrder(id+"-bid", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(price), matchingo.GTC, ""))
}

func main() {
	ob := matchingo.NewOrderBook(matchingo.WithCircuitBreaker(matchingo.FromInt(10))) // 10%

	trade(ob, "t1", 100)
	trade(ob, "t2", 105)
	fmt.Println("halted after 5% move:", ob.IsHalted())

	trade(ob, "t3", 80) // ~24% drop trips the breaker
	fmt.Println("halted after 24% move:", ob.IsHalted())

	_, err := ob.Process(matchingo.NewLimitOrder("late", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(80), matchingo.GTC, ""))
	fmt.Println("new order rejected:", errors.Is(err, matchingo.ErrTradingHalted))

	ob.Resume()
	_, err = ob.Process(matchingo.NewLimitOrder("late", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(80), matchingo.GTC, ""))
	fmt.Println("after Resume, accepted:", err == nil)
}
