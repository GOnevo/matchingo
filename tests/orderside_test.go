package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func TestOrderSideBid(t *testing.T) {
	orderSide := matchingo.NewOrderSideBid()

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
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
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Sell,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
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
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Sell,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
		"",
		"",
	)

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Sell,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(30),
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
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(30),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o4.Price(), o4.Quantity()) != true {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}

	o5 := matchingo.NewLimitOrder(
		"order-5",
		matchingo.Buy,
		fpdecimal.FromInt(20),
		fpdecimal.FromInt(20),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o5.Price(), o5.Quantity()) != true {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}

	o7 := matchingo.NewLimitOrder(
		"order-7",
		matchingo.Buy,
		fpdecimal.FromInt(31),
		fpdecimal.FromInt(100),
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
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
		"",
		"",
	)

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(30),
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
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(30),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o4.Price(), o4.Quantity()) != true {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o5 := matchingo.NewLimitOrder(
		"order-5",
		matchingo.Sell,
		fpdecimal.FromInt(20),
		fpdecimal.FromInt(20),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o5.Price(), o5.Quantity()) != true {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o6 := matchingo.NewLimitOrder(
		"order-6",
		matchingo.Sell,
		fpdecimal.FromInt(21),
		fpdecimal.FromInt(20),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o6.Price(), o6.Quantity()) != false {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o7 := matchingo.NewLimitOrder(
		"order-7",
		matchingo.Buy,
		fpdecimal.FromInt(31),
		fpdecimal.FromInt(100),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o7.Price(), o7.Quantity()) != false {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}
}

var BenchOrderSideBid = matchingo.NewOrderSideBid()

func BenchmarkOrderSide(b *testing.B) {
	stopwatch := time.Now()
	for i := 1; i < b.N; i++ {
		BenchOrderSideBid.Append(matchingo.NewLimitOrder(
			fmt.Sprintf("order-%d", i),
			matchingo.Buy,
			BenchQuantity,
			BenchPrice,
			"",
			"",
		))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("\n\nElapsed: %s\nTransactions per second: %f\n", elapsed, float64(b.N)/elapsed.Seconds())
}
