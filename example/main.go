package main

import (
	"fmt"
	"github.com/gonevo/matchingo"
	"github.com/shopspring/decimal"
)

func main() {
	orderBook := matchingo.NewOrderBook()
	fmt.Println(orderBook.Process(matchingo.NewLimitOrder("1", matchingo.Sell, decimal.New(10, 0), decimal.New(10, 0), "", "")))
	fmt.Println(orderBook.Process(matchingo.NewLimitOrder("2", matchingo.Buy, decimal.New(9, 0), decimal.New(10, 0), "", "")))
	fmt.Println(orderBook.Process(matchingo.NewMarketOrder("3", matchingo.Buy, decimal.New(10, 0))))
	fmt.Println(orderBook)
}
