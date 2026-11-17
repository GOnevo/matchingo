// Example 10 — Event Emitter
//
// The order book can push real-time events to any EventEmitter you attach.
// This decouples the engine from transport concerns (WebSocket, Redis, logs).
//
// Event types:
//
//	EventTrade     — a fill occurred (taker + maker order IDs, price, qty)
//	EventVolume    — the visible volume at a price level changed
//	EventDepth     — full depth snapshot was requested
//	EventCanceled  — an order was canceled
//	EventActivated — a stop-limit order became active
//
// Concepts shown:
//   - Implementing matchingo.EventEmitter
//   - ob.SetEmitter(e)
//   - Receiving typed event data via type-switch
package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

// logger is a trivial EventEmitter that prints every event to stdout.
type logger struct{}

func (l *logger) Emit(eventType matchingo.EventType, data any) {
	switch eventType {
	case matchingo.EventTrade:
		d := data.(matchingo.TradeEventData)
		fmt.Printf("  [TRADE]     order=%s  price=%s  amount=%s  role=%s\n",
			d.OrderID, d.Price, d.Amount, d.Role)

	case matchingo.EventVolume:
		d := data.(matchingo.VolumeEventData)
		fmt.Printf("  [VOLUME]    side=%s  price=%s  volume=%s\n",
			d.Side, d.Price, d.Volume)

	case matchingo.EventCanceled:
		fmt.Printf("  [CANCELED]  id=%s\n", data)

	case matchingo.EventActivated:
		fmt.Printf("  [ACTIVATED] id=%s\n", data)

	default:
		fmt.Printf("  [%s] %v\n", eventType, data)
	}
}

func main() {
	ob := matchingo.NewOrderBook()
	ob.SetEmitter(&logger{})

	fmt.Println("=== placing resting sell ===")
	ob.Process(matchingo.NewLimitOrder("sell-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(100), matchingo.GTC, ""))

	fmt.Println("\n=== matching buy ===")
	ob.Process(matchingo.NewLimitOrder("buy-1", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(100), matchingo.GTC, ""))

	fmt.Println("\n=== canceling leftover sell ===")
	ob.Cancel("sell-1")

	fmt.Println("\n=== requesting depth (triggers EventDepth) ===")
	ob.Depth()
}
