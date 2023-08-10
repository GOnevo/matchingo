package main

import (
	"fmt"

	"github.com/gonevo/matchingo"
)

func main() {
	orderBook := matchingo.NewOrderBook()
	matchingo.SetDecimalFraction(5) // default is 3

	done1, _ := orderBook.Process(matchingo.NewLimitOrder("order1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(10), "", ""))
	fmt.Println(done1)
	fmt.Println(orderBook)

	done2, _ := orderBook.Process(matchingo.NewLimitOrder("order2", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(10), "", ""))
	fmt.Println(done2)
	fmt.Println(orderBook)

	done3, _ := orderBook.Process(matchingo.NewMarketOrder("order3", matchingo.Buy, matchingo.FromFloat(5)))
	fmt.Println(done3)
	fmt.Println(orderBook)
}
