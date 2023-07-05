package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func main() {
	orderBook := matchingo.NewOrderBook()
	fmt.Println(orderBook.Process(matchingo.NewLimitOrder("1", matchingo.Sell, fpdecimal.FromInt(10), fpdecimal.FromInt(10), "", "")))
	fmt.Println(orderBook.Process(matchingo.NewLimitOrder("2", matchingo.Buy, fpdecimal.FromInt(9), fpdecimal.FromInt(10), "", "")))
	fmt.Println(orderBook.Process(matchingo.NewMarketOrder("3", matchingo.Buy, fpdecimal.FromInt(10))))
	fmt.Println(orderBook)
}
