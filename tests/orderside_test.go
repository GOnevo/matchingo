package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"

	"github.com/shopspring/decimal"
)

func TestOrderSideBid(t *testing.T) {
	orderSide := matchingo.NewOrderSideBid()

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(20, 0),
		"",
		"",
	)

	if orderSide.BestPriceQueue() != nil {
		t.Fatal("invalid price levels")
	}

	orderSide.Append(o1)
	orderSide.Append(o2)

	if orderSide.Depth() != 2 {
		t.Fatal("invalid depth")
	}

	if orderSide.Len() != 2 {
		t.Fatal("invalid orders count")
	}

	bestOrder := orderSide.BestPriceQueue().First()

	if orderSide.BestPriceQueue().Len() != 1 {
		t.Fatal("invalid best price queue size")
	}

	if bestOrder.ID() != o2.ID() {
		t.Fatal("invalid sorting")
	}

	prices := orderSide.Prices()

	if prices[0] != o2.Price() {
		t.Fatal("invalid price sorting for orderSide.Prices() slice")
	}

	if prices[1] != o1.Price() {
		t.Fatal("invalid price sorting for orderSide..Prices() slice")
	}

	if orderSide.String() == "" {
		t.Fatal("String not work")
	}
}

func TestOrderSideAsk(t *testing.T) {
	orderSide := matchingo.NewOrderSideAsk()

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(20, 0),
		"",
		"",
	)

	if orderSide.BestPriceQueue() != nil {
		t.Fatal("invalid price levels")
	}

	orderSide.Append(o1)
	orderSide.Append(o2)

	if orderSide.Depth() != 2 {
		t.Fatal("invalid depth")
	}

	if orderSide.Len() != 2 {
		t.Fatal("invalid orders count")
	}

	bestOrder := orderSide.BestPriceQueue().First()

	if bestOrder.ID() != o1.ID() {
		t.Fatal("invalid sorting")
	}

	if orderSide.BestPriceQueue().Len() != 1 {
		t.Fatal("invalid best price queue size")
	}

	prices := orderSide.Prices()

	if prices[0] != o1.Price() {
		t.Fatal("invalid price sorting for orderSide..Prices() slice")
	}

	if prices[1] != o2.Price() {
		t.Fatal("invalid price sorting for orderSide..Prices() slice")
	}

	if orderSide.String() == "" {
		t.Fatal("String not work")
	}
}

func TestOrderSide_CanBuyOrderBeFilled(t *testing.T) {
	orderSide := matchingo.NewOrderSideAsk()

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(20, 0),
		"",
		"",
	)

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(30, 0),
		"",
		"",
	)

	if orderSide.BestPriceQueue() != nil {
		t.Fatal("invalid price levels")
	}

	orderSide.Append(o1)
	orderSide.Append(o2)
	orderSide.Append(o3)

	if orderSide.Depth() != 3 {
		t.Fatal("invalid depth")
	}

	if orderSide.Len() != 3 {
		t.Fatal("invalid orders count")
	}

	o4 := matchingo.NewLimitOrder(
		"order-4",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(30, 0),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o4.Price(), o4.Quantity()) != true {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}

	o5 := matchingo.NewLimitOrder(
		"order-5",
		matchingo.Buy,
		decimal.New(20, 0),
		decimal.New(20, 0),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o5.Price(), o5.Quantity()) != true {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}

	o7 := matchingo.NewLimitOrder(
		"order-7",
		matchingo.Buy,
		decimal.New(31, 0),
		decimal.New(100, 0),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o7.Price(), o7.Quantity()) != false {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}
}

func TestOrderSide_CanSellOrderBeFilled(t *testing.T) {
	orderSide := matchingo.NewOrderSideBid()

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(20, 0),
		"",
		"",
	)

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(30, 0),
		"",
		"",
	)

	if orderSide.BestPriceQueue() != nil {
		t.Fatal("invalid price levels")
	}

	orderSide.Append(o1)
	orderSide.Append(o2)
	orderSide.Append(o3)

	if orderSide.Depth() != 3 {
		t.Fatal("invalid depth")
	}

	if orderSide.Len() != 3 {
		t.Fatal("invalid orders count")
	}

	o4 := matchingo.NewLimitOrder(
		"order-4",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(30, 0),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o4.Price(), o4.Quantity()) != true {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o5 := matchingo.NewLimitOrder(
		"order-5",
		matchingo.Sell,
		decimal.New(20, 0),
		decimal.New(20, 0),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o5.Price(), o5.Quantity()) != true {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o6 := matchingo.NewLimitOrder(
		"order-6",
		matchingo.Sell,
		decimal.New(21, 0),
		decimal.New(20, 0),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o6.Price(), o6.Quantity()) != false {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o7 := matchingo.NewLimitOrder(
		"order-7",
		matchingo.Buy,
		decimal.New(31, 0),
		decimal.New(100, 0),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o7.Price(), o7.Quantity()) != false {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}
}

func BenchmarkOrderSide(b *testing.B) {
	ot := matchingo.NewOrderSideBid()
	stopwatch := time.Now()
	price := decimal.New(10, 0)

	for i := 0; i < b.N; i++ {
		ot.Append(matchingo.NewLimitOrder(
			fmt.Sprintf("order-%d", i),
			matchingo.Buy,
			price,
			decimal.New(int64(i), 0),
			"",
			"",
		))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("\n\nElapsed: %s\nTransactions per second: %f\n", elapsed, float64(b.N)/elapsed.Seconds())
}
