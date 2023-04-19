package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
	"github.com/shopspring/decimal"
)

func addDepth(ob *matchingo.OrderBook, prefix string, quantity decimal.Decimal) {
	for i := 50; i < 100; i = i + 10 {
		ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("%sbuy-%d", prefix, i), matchingo.Buy, quantity, decimal.New(int64(i), 0), "", ""))
	}

	for i := 100; i < 150; i = i + 10 {
		ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("%ssell-%d", prefix, i), matchingo.Sell, quantity, decimal.New(int64(i), 0), "", ""))
	}
}

func TestMarketQuantityQuoteProcessing(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("order-1", matchingo.Sell, decimal.New(10, 0), decimal.New(10, 0), "", ""))

	done, err := ob.Process(matchingo.NewMarketOrder("order-2", matchingo.Buy, decimal.New(100, 0)))
	if err != nil {
		t.Fatal(err)
	}

	if done.Trade.Order.ID() != "order-2" {
		t.Fatal("Wrong order id")
	}

	if done.Trade.Orders["order-1"] == nil {
		t.Fatal("Wrong orders id")
	}

	if done.Trade.Orders["order-1"].Quantity.Equal(decimal.New(10, 0)) == false {
		t.Fatal("Wrong orders quantity")
	}

	if done.Left.Equal(decimal.New(0, 0)) != true {
		t.Fatal("Wrong quote calculation")
	}
}

func TestLimitFOKProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()

	addDepth(ob, "", decimal.New(2, 0))

	done, err := ob.Process(matchingo.NewLimitOrder("order-b100", matchingo.Buy, decimal.New(11, 0), decimal.New(100, 0), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Trade.Order.ID() != "order-b100" {
		t.Fatal("Wrong done id")
	}

	if !done.Trade.Order.IsCanceled() {
		t.Fatal("Wrong done canceled")
	}

	if !done.Left.Equal(decimal.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(decimal.Zero) {
		t.Fatal("Wrong quantity processed")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-b100", matchingo.Sell, decimal.New(11, 0), decimal.New(100, 0), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Trade.Order.ID() != "order-b100" {
		t.Fatal("Wrong done id")
	}

	if !done.Trade.Order.IsCanceled() {
		t.Fatal("Wrong done canceled")
	}

	if !done.Left.Equal(decimal.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(decimal.Zero) {
		t.Fatal("Wrong quantity processed")
	}
}

func TestLimitIOCProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", decimal.New(2, 0))

	done, err := ob.Process(matchingo.NewLimitOrder("order-ioc", matchingo.Buy, decimal.New(11, 0), decimal.New(200, 0), matchingo.IOC, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Trade.Order.ID() != "order-ioc" {
		t.Fatal("Wrong done id")
	}

	if len(done.Canceled) != 1 {
		t.Fatal("Wrong canceled")
	}

	if !done.Left.Equal(decimal.New(1, 0)) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(decimal.New(10, 0)) {
		t.Fatal("Wrong quantity processed")
	}
}

func TestLimitPlace(t *testing.T) {
	ob := matchingo.NewOrderBook()
	quantity := decimal.New(2, 0)
	for i := 50; i < 100; i = i + 10 {
		done, err := ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("buy-%d", i), matchingo.Buy, quantity, decimal.New(int64(i), 0), "", ""))
		if len(done.Trade.Orders) != 0 {
			t.Fatal("OrderBook failed to process limit order (participants is not empty)")
		}
		if done.Stored == false {
			t.Fatal("OrderBook failed to process limit order (stores)")
		}
		if done.Left.Sign() != 0 {
			t.Fatal("OrderBook failed to process limit order (left)")
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 100; i < 150; i = i + 10 {
		done, err := ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("sell-%d", i), matchingo.Sell, quantity, decimal.New(int64(i), 0), "", ""))
		if len(done.Trade.Orders) != 0 {
			t.Fatal("OrderBook failed to process limit order (participants is not empty)")
		}
		if done.Stored == false {
			t.Fatal("OrderBook failed to process limit order (stores)")
		}
		if done.Left.Sign() != 0 {
			t.Fatal("OrderBook failed to process limit order (left)")
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLimitProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", decimal.New(2, 0))

	done, err := ob.Process(matchingo.NewLimitOrder("order-b100", matchingo.Buy, decimal.New(1, 0), decimal.New(100, 0), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Trade.Order.ID() != "order-b100" {
		t.Fatal("Wrong order id")
	}

	if done.Stored {
		t.Fatal("Wrong stored")
	}

	if !done.Processed.Equal(decimal.New(1, 0)) {
		t.Fatal("Wrong quantity processed")
	}

	if !done.Left.Equal(decimal.Zero) {
		t.Fatal("Wrong partial quantity left")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-b150", matchingo.Buy, decimal.New(10, 0), decimal.New(150, 0), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trade.Orders) != 5 {
		t.Fatal("Wrong participants count")
	}

	if !done.Stored {
		t.Fatal("Wrong stored")
	}

	if done.Trade.Order.ID() != "order-b150" {
		t.Fatal("Wrong order id")
	}

	if !done.Processed.Equal(decimal.New(9, 0)) {
		t.Fatal("Wrong partial quantity processed", done.Processed)
	}

	if _, err := ob.Process(matchingo.NewLimitOrder("buy-70", matchingo.Sell, decimal.New(11, 0), decimal.New(40, 0), "", "")); err == nil {
		t.Fatal("Can add existing order")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-s40", matchingo.Sell, decimal.New(11, 0), decimal.New(40, 0), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trade.Orders) != 6 {
		t.Fatal("Wrong participants count")
	}

	if done.Left.Sign() != 0 {
		t.Fatal("Wrong left")
	}
}

func TestOCOProcessStop(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("oco-1", matchingo.Buy, decimal.New(1, 0), decimal.New(100, 0), "", "oco-2"))
	ob.Process(
		matchingo.NewStopLimitOrder("oco-2", matchingo.Buy, decimal.New(1, 0), decimal.New(100, 0), decimal.New(101, 0), "oco-1"),
	)

	if ob.Stop.Len() != 1 {
		t.Fatal("Wrong stop book")
	}

	done, err := ob.Process(matchingo.NewLimitOrder("simple-1", matchingo.Sell, decimal.New(1, 0), decimal.New(100, 0), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Trade.Order.ID() != "simple-1" {
		t.Fatal("Wrong order id")
	}

	if ob.Stop.Len() != 0 {
		t.Fatal("Wrong stop book")
	}

	if done.Canceled[0] != "oco-2" {
		t.Fatal("Wrong canceled")
	}
}

func TestOCOProcessLimit(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(
		matchingo.NewStopLimitOrder("oco-2", matchingo.Buy, decimal.New(1, 0), decimal.New(150, 0), decimal.New(101, 0), "oco-1"),
	)
	ob.Process(matchingo.NewLimitOrder("o1", matchingo.Sell, decimal.New(1, 0), decimal.New(101, 0), "", ""))
	ob.Process(matchingo.NewLimitOrder("o2", matchingo.Buy, decimal.New(1, 0), decimal.New(101, 0), "", ""))

	ob.Process(matchingo.NewLimitOrder("oco-1", matchingo.Buy, decimal.New(1, 0), decimal.New(100, 0), "", "oco-2"))

	done, err := ob.Process(matchingo.NewLimitOrder("simple-1", matchingo.Sell, decimal.New(1, 0), decimal.New(100, 0), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Trade.Order.ID() != "simple-1" {
		t.Fatal("Wrong order id")
	}

	if done.Trade.Orders["oco-2"] == nil {
		t.Fatal("Wrong orders id")
	}

	if len(done.Canceled) == 0 {
		t.Fatal("Wrong canceled slice")
	}

	if done.Canceled[0] != "oco-1" {
		t.Fatal("Wrong canceled")
	}

	if len(ob.OCO) != 0 {
		t.Fatal("Wrong oco book")
	}
}

func TestMarketProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", decimal.New(2, 0))

	done, err := ob.Process(
		matchingo.NewMarketOrder("order-buy-3", matchingo.Buy, decimal.New(300, 0)),
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trade.Orders) != 2 {
		t.Fatal("Invalid participants length")
	}

	if !done.Left.Equal(decimal.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(decimal.New(300, 0)) {
		t.Fatal("Wrong quantity processed")
	}

	if done.Trade.Order.IsCanceled() {
		t.Fatal("order is not canceled")
	}

	done, err = ob.Process(matchingo.NewMarketOrder("order-sell-12", matchingo.Sell, decimal.New(12, 0)))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trade.Orders) != 5 {
		t.Fatal("Invalid participants length")
	}

	if !done.Left.Equal(decimal.New(2, 0)) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(decimal.New(10, 0)) {
		t.Fatal("Wrong quantity processed")
	}

	if !done.Trade.Order.IsCanceled() {
		t.Fatal("order is not canceled")
	}
}

func TestStopOrderProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()

	stop := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(10, 0),
		decimal.New(10, 0),

		"",
	)

	ob.Process(stop)

	if ob.Stop.Len() != 1 {
		t.Fatal("stop book is broken")
	}

	ob.Process(matchingo.NewLimitOrder("order-limit-1", matchingo.Buy, decimal.New(10, 0), decimal.New(10, 0), "", ""))
	ob.Process(matchingo.NewLimitOrder("order-limit-2", matchingo.Sell, decimal.New(10, 0), decimal.New(10, 0), "", ""))

	if ob.Stop.Len() != 0 {
		t.Fatal("stop book is broken")
	}
}

func TestPriceCalculation(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "05-", decimal.New(10, 0))
	addDepth(ob, "10-", decimal.New(10, 0))
	addDepth(ob, "15-", decimal.New(10, 0))

	price, err := ob.CalculateMarketPrice(matchingo.Buy, decimal.New(115, 0))
	if err != nil {
		t.Fatal(err)
	}

	if !price.Equal(decimal.New(13150, 0)) {
		t.Fatal("invalid price", price)
	}

	price, err = ob.CalculateMarketPrice(matchingo.Buy, decimal.New(200, 0))
	if err == nil {
		t.Fatal("invalid quantity count")
	}

	if !price.Equal(decimal.New(18000, 0)) {
		t.Fatal("invalid price", price)
	}

	// -------

	price, err = ob.CalculateMarketPrice(matchingo.Sell, decimal.New(115, 0))
	if err != nil {
		t.Fatal(err)
	}

	if !price.Equal(decimal.New(8700, 0)) {
		t.Fatal("invalid price", price)
	}

	price, err = ob.CalculateMarketPrice(matchingo.Sell, decimal.New(200, 0))
	if err == nil {
		t.Fatal("invalid quantity count")
	}

	if !price.Equal(decimal.New(10500, 0)) {
		t.Fatal("invalid price", price)
	}
}

func BenchmarkAppendLimitOrders(b *testing.B) {
	ob := matchingo.NewOrderBook()
	stopwatch := time.Now()
	for i := 0; i < b.N; i++ {
		ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("buy-%d", i), matchingo.Buy, decimal.New(16, 0), decimal.New(10, 0), "", ""))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("Elapsed: %s\nTransactions per second (avg): %f\n", elapsed, float64(b.N*2)/elapsed.Seconds())
}

func BenchmarkLimitOrders(b *testing.B) {
	ob := matchingo.NewOrderBook()
	stopwatch := time.Now()
	for i := 0; i < b.N; i++ {
		ob.Process(matchingo.NewLimitOrder("order-buy", matchingo.Buy, decimal.New(16, 0), decimal.New(150, 0), "", ""))
	}
	for i := 0; i < b.N; i++ {
		ob.Process(matchingo.NewLimitOrder("order-sell", matchingo.Sell, decimal.New(16, 0), decimal.New(150, 0), "", ""))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("Elapsed: %s\nTransactions per second (avg): %f\n", elapsed, float64(b.N*2)/elapsed.Seconds())
}
