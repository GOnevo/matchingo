package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func addDepth(ob *matchingo.OrderBook, prefix string, quantity fpdecimal.Decimal) {
	for i := 50; i < 100; i = i + 10 {
		ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("%sbuy-%d", prefix, i), matchingo.Buy, quantity, fpdecimal.FromInt(int64(i)), "", ""))
	}

	for i := 100; i < 150; i = i + 10 {
		ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("%ssell-%d", prefix, i), matchingo.Sell, quantity, fpdecimal.FromInt(int64(i)), "", ""))
	}
}

func TestMarketQuantityQuoteProcessing(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("order-1", matchingo.Sell, fpdecimal.FromInt(10), fpdecimal.FromInt(10), "", ""))

	done, err := ob.Process(matchingo.NewMarketQuoteOrder("order-2", matchingo.Buy, fpdecimal.FromInt(100)))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-2" {
		t.Fatal("Wrong order id")
	}

	if done.GetTradeOrder("order-1") == nil {
		t.Fatal("Wrong orders id")
	}

	if done.GetTradeOrder("order-1").Quantity.Equal(fpdecimal.FromInt(10)) == false {
		t.Fatal("Wrong orders quantity")
	}

	if done.Left.Equal(fpdecimal.FromInt(0)) != true {
		t.Fatal("Wrong quote calculation")
	}
}

func TestLimitFOKProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()

	addDepth(ob, "", fpdecimal.FromInt(2))

	done, err := ob.Process(matchingo.NewLimitOrder("order-b100", matchingo.Buy, fpdecimal.FromInt(11), fpdecimal.FromInt(100), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-b100" {
		t.Fatal("Wrong done id")
	}

	if !done.Order.IsCanceled() {
		t.Fatal("Wrong done canceled")
	}

	if !done.Left.Equal(fpdecimal.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(fpdecimal.Zero) {
		t.Fatal("Wrong quantity processed")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-s100", matchingo.Sell, fpdecimal.FromInt(11), fpdecimal.FromInt(100), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-s100" {
		t.Fatal("Wrong done id")
	}

	if !done.Order.IsCanceled() {
		t.Fatal("Wrong done canceled")
	}

	if !done.Left.Equal(fpdecimal.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(fpdecimal.Zero) {
		t.Fatal("Wrong quantity processed")
	}
}

func TestLimitIOCProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", fpdecimal.FromInt(2))

	done, err := ob.Process(matchingo.NewLimitOrder("order-ioc", matchingo.Buy, fpdecimal.FromInt(11), fpdecimal.FromInt(200), matchingo.IOC, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-ioc" {
		t.Fatal("Wrong done id")
	}

	if len(done.Canceled) != 1 {
		t.Fatal("Wrong canceled")
	}

	if !done.Left.Equal(fpdecimal.FromInt(1)) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(fpdecimal.FromInt(10)) {
		t.Fatal("Wrong quantity processed")
	}

	if done.String() == "" {
		t.Fatal("Wrong to JSON")
	}
}

func TestLimitPlace(t *testing.T) {
	ob := matchingo.NewOrderBook()
	quantity := fpdecimal.FromInt(2)
	for i := 50; i < 100; i = i + 10 {
		done, err := ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("buy-%d", i), matchingo.Buy, quantity, fpdecimal.FromInt(int64(i)), "", ""))
		if len(done.Trades) != 0 {
			t.Fatal("OrderBook failed to process limit order (participants is not empty)")
		}
		if done.Stored == false {
			t.Fatal("OrderBook failed to process limit order (stores)")
		}
		if done.Left.Equal(fpdecimal.Zero) != true {
			t.Fatal("OrderBook failed to process limit order (left)")
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 100; i < 150; i = i + 10 {
		done, err := ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("sell-%d", i), matchingo.Sell, quantity, fpdecimal.FromInt(int64(i)), "", ""))
		if len(done.Trades) != 0 {
			t.Fatal("OrderBook failed to process limit order (participants is not empty)")
		}
		if done.Stored == false {
			t.Fatal("OrderBook failed to process limit order (stores)")
		}
		if done.Left.Equal(fpdecimal.Zero) != true {
			t.Fatal("OrderBook failed to process limit order (left)")
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLimitProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", fpdecimal.FromInt(2))

	done, err := ob.Process(matchingo.NewLimitOrder("order-b100", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(100), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-b100" {
		t.Fatal("Wrong order id")
	}

	if done.Stored {
		t.Fatal("Wrong stored")
	}

	if !done.Processed.Equal(fpdecimal.FromInt(1)) {
		t.Fatal("Wrong quantity processed")
	}

	if !done.Left.Equal(fpdecimal.Zero) {
		t.Fatal("Wrong partial quantity left")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-b150", matchingo.Buy, fpdecimal.FromInt(10), fpdecimal.FromInt(150), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trades) != 6 {
		t.Fatal("Wrong participants count")
	}

	if !done.Stored {
		t.Fatal("Wrong stored")
	}

	if done.Order.ID() != "order-b150" {
		t.Fatal("Wrong order id")
	}

	if !done.Processed.Equal(fpdecimal.FromInt(9)) {
		t.Fatal("Wrong partial quantity processed", done.Processed)
	}

	if _, err := ob.Process(matchingo.NewLimitOrder("buy-70", matchingo.Sell, fpdecimal.FromInt(11), fpdecimal.FromInt(40), "", "")); err == nil {
		t.Fatal("Can add existing order")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-s40", matchingo.Sell, fpdecimal.FromInt(11), fpdecimal.FromInt(40), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trades) != 7 {
		t.Fatal("Wrong participants count")
	}

	if done.Left.Equal(fpdecimal.Zero) != true {
		t.Fatal("Wrong left")
	}
}

func TestOCOProcessStop(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("oco-1", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(100), "", "oco-2"))
	ob.Process(
		matchingo.NewStopLimitOrder("oco-2", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(100), fpdecimal.FromInt(101), "oco-1"),
	)

	if ob.Stop.Len() != 1 {
		t.Fatal("Wrong stop book")
	}

	done, err := ob.Process(matchingo.NewLimitOrder("simple-1", matchingo.Sell, fpdecimal.FromInt(1), fpdecimal.FromInt(100), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "simple-1" {
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
		matchingo.NewStopLimitOrder("oco-2", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(150), fpdecimal.FromInt(101), "oco-1"),
	)
	ob.Process(matchingo.NewLimitOrder("o1", matchingo.Sell, fpdecimal.FromInt(1), fpdecimal.FromInt(101), "", ""))
	ob.Process(matchingo.NewLimitOrder("o2", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(101), "", ""))

	ob.Process(matchingo.NewLimitOrder("oco-1", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(100), "", "oco-2"))

	done, err := ob.Process(matchingo.NewLimitOrder("simple-1", matchingo.Sell, fpdecimal.FromInt(1), fpdecimal.FromInt(100), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "simple-1" {
		t.Fatal("Wrong order id")
	}

	if done.GetTradeOrder("oco-2") == nil {
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
	addDepth(ob, "", fpdecimal.FromInt(2))

	done, err := ob.Process(
		matchingo.NewMarketQuoteOrder("order-buy-3", matchingo.Buy, fpdecimal.FromInt(300)),
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trades) != 3 {
		t.Fatal("Invalid participants length")
	}

	if !done.Left.Equal(fpdecimal.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(fpdecimal.FromInt(300)) {
		t.Fatal("Wrong quantity processed")
	}

	if done.Order.IsCanceled() {
		t.Fatal("order is not canceled")
	}

	done, err = ob.Process(matchingo.NewMarketOrder("order-sell-12", matchingo.Sell, fpdecimal.FromInt(12)))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trades) != 6 {
		t.Fatal("Invalid participants length")
	}

	if !done.Left.Equal(fpdecimal.FromInt(2)) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(fpdecimal.FromInt(10)) {
		t.Fatal("Wrong quantity processed")
	}

	if !done.Order.IsCanceled() {
		t.Fatal("order is not canceled")
	}
}

func TestStopOrderProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()

	stop := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),

		"",
	)

	ob.Process(stop)

	if ob.Stop.Len() != 1 {
		t.Fatal("stop book is broken")
	}

	ob.Process(matchingo.NewLimitOrder("order-limit-1", matchingo.Buy, fpdecimal.FromInt(10), fpdecimal.FromInt(10), "", ""))
	ob.Process(matchingo.NewLimitOrder("order-limit-2", matchingo.Sell, fpdecimal.FromInt(10), fpdecimal.FromInt(10), "", ""))

	if ob.Stop.Len() != 0 {
		t.Fatal("stop book is broken")
	}
}

func TestPriceCalculation(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "05-", fpdecimal.FromInt(10))
	addDepth(ob, "10-", fpdecimal.FromInt(10))
	addDepth(ob, "15-", fpdecimal.FromInt(10))

	price, err := ob.CalculateMarketPrice(matchingo.Buy, fpdecimal.FromInt(115))
	if err != nil {
		t.Fatal(err)
	}

	if !price.Equal(fpdecimal.FromInt(13150)) {
		t.Fatal("invalid price", price)
	}

	price, err = ob.CalculateMarketPrice(matchingo.Buy, fpdecimal.FromInt(200))
	if err == nil {
		t.Fatal("invalid quantity count")
	}

	if !price.Equal(fpdecimal.FromInt(18000)) {
		t.Fatal("invalid price", price)
	}

	// -------

	price, err = ob.CalculateMarketPrice(matchingo.Sell, fpdecimal.FromInt(115))
	if err != nil {
		t.Fatal(err)
	}

	if !price.Equal(fpdecimal.FromInt(8700)) {
		t.Fatal("invalid price", price)
	}

	price, err = ob.CalculateMarketPrice(matchingo.Sell, fpdecimal.FromInt(200))
	if err == nil {
		t.Fatal("invalid quantity count")
	}

	if !price.Equal(fpdecimal.FromInt(10500)) {
		t.Fatal("invalid price", price)
	}
}

var benchOb = matchingo.NewOrderBook()
var BenchPrice = fpdecimal.FromInt(16)
var BenchQuantity = fpdecimal.FromInt(150)

func BenchmarkAppendLimitOrders(b *testing.B) {
	stopwatch := time.Now()
	for i := 0; i < b.N; i++ {
		benchOb.Process(matchingo.NewLimitOrder(fmt.Sprintf("buy-%d", i), matchingo.Buy, BenchQuantity, BenchPrice, "", ""))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("Elapsed: %s\nTransactions per second (avg): %f\n", elapsed, float64(b.N*2)/elapsed.Seconds())
}

func BenchmarkLimitOrders(b *testing.B) {
	stopwatch := time.Now()
	for i := 0; i < b.N; i++ {
		benchOb.Process(matchingo.NewLimitOrder(fmt.Sprintf("sell-%d", i), matchingo.Sell, BenchQuantity, BenchPrice, "", ""))
	}
	for i := 0; i < b.N; i++ {
		benchOb.Process(matchingo.NewLimitOrder(fmt.Sprintf("buy-%d", i), matchingo.Buy, BenchQuantity, BenchPrice, "", ""))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("Elapsed: %s\nTransactions per second (avg): %f\n", elapsed, float64(b.N*2)/elapsed.Seconds())
}
