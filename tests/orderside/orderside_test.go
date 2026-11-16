package orderside

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
)

func TestOrderSide_Bid(t *testing.T) {
	orderSide := matchingo.NewOrderSideBid()

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
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

func TestOrderSide_Ask(t *testing.T) {
	orderSide := matchingo.NewOrderSideAsk()

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Sell,
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Sell,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
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
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Sell,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
		"",
		"",
	)

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Sell,
		matchingo.FromInt(10),
		matchingo.FromInt(30),
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
		matchingo.FromInt(10),
		matchingo.FromInt(30),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o4.Price(), o4.Quantity()) != true {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}

	o5 := matchingo.NewLimitOrder(
		"order-5",
		matchingo.Buy,
		matchingo.FromInt(20),
		matchingo.FromInt(20),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o5.Price(), o5.Quantity()) != true {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}

	o7 := matchingo.NewLimitOrder(
		"order-7",
		matchingo.Buy,
		matchingo.FromInt(31),
		matchingo.FromInt(100),
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
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
		"",
		"",
	)

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(30),
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
		matchingo.FromInt(10),
		matchingo.FromInt(30),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o4.Price(), o4.Quantity()) != true {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o5 := matchingo.NewLimitOrder(
		"order-5",
		matchingo.Sell,
		matchingo.FromInt(20),
		matchingo.FromInt(20),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o5.Price(), o5.Quantity()) != true {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o6 := matchingo.NewLimitOrder(
		"order-6",
		matchingo.Sell,
		matchingo.FromInt(21),
		matchingo.FromInt(20),
		"",
		"",
	)

	if orderSide.CanSellOrderBeFilled(o6.Price(), o6.Quantity()) != false {
		t.Fatal("invalid CanSellOrderBeFilled result")
	}

	o7 := matchingo.NewLimitOrder(
		"order-7",
		matchingo.Buy,
		matchingo.FromInt(31),
		matchingo.FromInt(100),
		"",
		"",
	)

	if orderSide.CanBuyOrderBeFilled(o7.Price(), o7.Quantity()) != false {
		t.Fatal("invalid CanBuyOrderBeFilled result")
	}
}

func TestOrderSide_CanOrderBeFilled(t *testing.T) {
	askSide := matchingo.NewOrderSideAsk()
	askSide.Append(matchingo.NewLimitOrder("ask-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	askSide.Append(matchingo.NewLimitOrder("ask-2", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(110), "", ""))

	if !askSide.CanOrderBeFilled(matchingo.Buy, matchingo.FromInt(110), matchingo.FromInt(10)) {
		t.Fatal("buy order should be fillable on ask side")
	}
	if askSide.CanOrderBeFilled(matchingo.Buy, matchingo.FromInt(110), matchingo.FromInt(20)) {
		t.Fatal("buy order should not be fillable with insufficient quantity")
	}

	bidSide := matchingo.NewOrderSideBid()
	bidSide.Append(matchingo.NewLimitOrder("bid-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	bidSide.Append(matchingo.NewLimitOrder("bid-2", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(90), "", ""))

	if !bidSide.CanOrderBeFilled(matchingo.Sell, matchingo.FromInt(90), matchingo.FromInt(10)) {
		t.Fatal("sell order should be fillable on bid side")
	}
	if bidSide.CanOrderBeFilled(matchingo.Sell, matchingo.FromInt(90), matchingo.FromInt(20)) {
		t.Fatal("sell order should not be fillable with insufficient quantity")
	}
}

func TestOrderSide_CanOrderBeFilled_Panic(t *testing.T) {
	askSide := matchingo.NewOrderSideAsk()
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("CanOrderBeFilled should panic on unrecognized side")
			}
		}()
		askSide.CanOrderBeFilled(matchingo.Side(99), matchingo.FromInt(100), matchingo.FromInt(10))
	}()
}

func TestOrderSide_CanBuyOrderBeFilled_Empty(t *testing.T) {
	askSide := matchingo.NewOrderSideAsk()
	if askSide.CanBuyOrderBeFilled(matchingo.FromInt(100), matchingo.FromInt(10)) {
		t.Fatal("empty side should not be fillable")
	}
}

func TestOrderSide_CanSellOrderBeFilled_Empty(t *testing.T) {
	bidSide := matchingo.NewOrderSideBid()
	if bidSide.CanSellOrderBeFilled(matchingo.FromInt(100), matchingo.FromInt(10)) {
		t.Fatal("empty side should not be fillable")
	}
}

func TestOrderSide_NextLevel(t *testing.T) {
	orderSide := matchingo.NewOrderSideAsk()

	if orderSide.NextLevel(matchingo.FromInt(100)) != nil {
		t.Fatal("NextLevel on empty side should return nil")
	}

	orderSide.Append(matchingo.NewLimitOrder("ask-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	orderSide.Append(matchingo.NewLimitOrder("ask-2", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(110), "", ""))
	orderSide.Append(matchingo.NewLimitOrder("ask-3", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(120), "", ""))

	next := orderSide.NextLevel(matchingo.FromInt(100))
	if next == nil {
		t.Fatal("NextLevel should return the next price level")
	}
	if !next.Price().Equal(matchingo.FromInt(110)) {
		t.Fatal("NextLevel should return price 110")
	}

	next = orderSide.NextLevel(matchingo.FromInt(110))
	if next == nil {
		t.Fatal("NextLevel should return the next price level")
	}
	if !next.Price().Equal(matchingo.FromInt(120)) {
		t.Fatal("NextLevel should return price 120")
	}

	next = orderSide.NextLevel(matchingo.FromInt(120))
	if next != nil {
		t.Fatal("NextLevel past last level should return nil")
	}
}

var BenchPrice = matchingo.FromInt(16)
var BenchQuantity = matchingo.FromInt(150)

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
