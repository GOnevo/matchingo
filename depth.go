package matchingo

import (
	"encoding/json"
	"fmt"

	"github.com/nikolaydubina/fpdecimal"
)

// Level contains Price and Volume in depth
type Level struct {
	Price  fpdecimal.Decimal
	Volume fpdecimal.Decimal
}

type Depth struct {
	Ask map[string]string `json:"ask"`
	Bid map[string]string `json:"bid"`
}

// Depth returns Price levels and volume at Price level
func (ob *OrderBook) Depth() *Depth {
	var level *OrderQueue

	depth := &Depth{
		Ask: map[string]string{},
		Bid: map[string]string{},
	}

	level = ob.asks.BestPriceQueue()
	fmt.Println(level)
	for level != nil {
		depth.Ask[level.Price().String()] = level.Volume().String()
		level = ob.asks.NextLevel(level.Price())
	}

	level = ob.bids.BestPriceQueue()
	fmt.Println(level)
	for level != nil {
		depth.Bid[level.Price().String()] = level.Volume().String()
		level = ob.bids.NextLevel(level.Price())
	}
	return depth
}

// DepthJSON returns ask/bid depth as JSON
func (ob *OrderBook) DepthJSON() string {
	depth := ob.Depth()

	j, _ := json.Marshal(depth)

	return string(j)
}
