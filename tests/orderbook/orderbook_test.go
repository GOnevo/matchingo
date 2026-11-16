package orderbook

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
)

func addDepth(ob *matchingo.OrderBook, prefix string, quantity matchingo.Decimal) {
	for i := 50; i < 100; i = i + 10 {
		ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("%sbuy-%d", prefix, i), matchingo.Buy, quantity, matchingo.FromInt(int(i)), "", ""))
	}

	for i := 100; i < 150; i = i + 10 {
		ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("%ssell-%d", prefix, i), matchingo.Sell, quantity, matchingo.FromInt(int(i)), "", ""))
	}
}

func TestOrderBook_MarketQuoteProcessing(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("order-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(10), "", ""))

	done, err := ob.Process(matchingo.NewMarketQuoteOrder("order-2", matchingo.Buy, matchingo.FromInt(100)))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-2" {
		t.Fatal("Wrong order id")
	}

	if done.GetTradeOrder("order-1") == nil {
		t.Fatal("Wrong orders id")
	}

	if done.GetTradeOrder("order-1").Quantity.Equal(matchingo.FromInt(10)) == false {
		t.Fatal("Wrong orders quantity")
	}

	if done.Left.Equal(matchingo.FromInt(0)) != true {
		t.Fatal("Wrong quote calculation")
	}
}

func TestOrderBook_LimitFOKProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()

	addDepth(ob, "", matchingo.FromInt(2))

	done, err := ob.Process(matchingo.NewLimitOrder("order-b100", matchingo.Buy, matchingo.FromInt(11), matchingo.FromInt(100), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-b100" {
		t.Fatal("Wrong done id")
	}

	if !done.Order.IsCanceled() {
		t.Fatal("Wrong done canceled")
	}

	if !done.Left.Equal(matchingo.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(matchingo.Zero) {
		t.Fatal("Wrong quantity processed")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-s100", matchingo.Sell, matchingo.FromInt(11), matchingo.FromInt(100), matchingo.FOK, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-s100" {
		t.Fatal("Wrong done id")
	}

	if !done.Order.IsCanceled() {
		t.Fatal("Wrong done canceled")
	}

	if !done.Left.Equal(matchingo.Zero) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(matchingo.Zero) {
		t.Fatal("Wrong quantity processed")
	}
}

func TestOrderBook_LimitIOCProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", matchingo.FromInt(2))

	done, err := ob.Process(matchingo.NewLimitOrder("order-ioc", matchingo.Buy, matchingo.FromInt(11), matchingo.FromInt(200), matchingo.IOC, ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-ioc" {
		t.Fatal("Wrong done id")
	}

	if len(done.Canceled) != 1 {
		t.Fatal("Wrong canceled")
	}

	if !done.Left.Equal(matchingo.FromInt(1)) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(matchingo.FromInt(10)) {
		t.Fatal("Wrong quantity processed")
	}

	if done.String() == "" {
		t.Fatal("Wrong to JSON")
	}
}

func TestOrderBook_LimitPlace(t *testing.T) {
	ob := matchingo.NewOrderBook()
	quantity := matchingo.FromInt(2)
	for i := 50; i < 100; i = i + 10 {
		done, err := ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("buy-%d", i), matchingo.Buy, quantity, matchingo.FromInt(int(i)), "", ""))
		if len(done.Trades) != 0 {
			t.Fatal("OrderBook failed to process limit order (participants is not empty)")
		}
		if done.Stored == false {
			t.Fatal("OrderBook failed to process limit order (stores)")
		}
		if done.Left.Equal(matchingo.Zero) != true {
			t.Fatal("OrderBook failed to process limit order (left)")
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 100; i < 150; i = i + 10 {
		done, err := ob.Process(matchingo.NewLimitOrder(fmt.Sprintf("sell-%d", i), matchingo.Sell, quantity, matchingo.FromInt(int(i)), "", ""))
		if len(done.Trades) != 0 {
			t.Fatal("OrderBook failed to process limit order (participants is not empty)")
		}
		if done.Stored == false {
			t.Fatal("OrderBook failed to process limit order (stores)")
		}
		if done.Left.Equal(matchingo.Zero) != true {
			t.Fatal("OrderBook failed to process limit order (left)")
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestOrderBook_LimitProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", matchingo.FromInt(2))

	done, err := ob.Process(matchingo.NewLimitOrder("order-b100", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if done.Order.ID() != "order-b100" {
		t.Fatal("Wrong order id")
	}

	if done.Stored {
		t.Fatal("Wrong stored")
	}

	if !done.Processed.Equal(matchingo.FromInt(1)) {
		t.Fatal("Wrong quantity processed")
	}

	if !done.Left.Equal(matchingo.Zero) {
		t.Fatal("Wrong partial quantity left")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-b150", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(150), "", ""))
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

	if !done.Processed.Equal(matchingo.FromInt(9)) {
		t.Fatal("Wrong partial quantity processed", done.Processed)
	}

	if _, err := ob.Process(matchingo.NewLimitOrder("buy-70", matchingo.Sell, matchingo.FromInt(11), matchingo.FromInt(40), "", "")); err == nil {
		t.Fatal("Can add existing order")
	}

	done, err = ob.Process(matchingo.NewLimitOrder("order-s40", matchingo.Sell, matchingo.FromInt(11), matchingo.FromInt(40), "", ""))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trades) != 7 {
		t.Fatal("Wrong participants count")
	}

	if done.Left.Equal(matchingo.Zero) != true {
		t.Fatal("Wrong left")
	}
}

func TestOrderBook_OCOProcessStop(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("oco-1", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), "", "oco-2"))
	ob.Process(
		matchingo.NewStopLimitOrder("oco-2", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), matchingo.FromInt(101), "oco-1"),
	)

	if ob.Stop.Len() != 1 {
		t.Fatal("Wrong stop book")
	}

	done, err := ob.Process(matchingo.NewLimitOrder("simple-1", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(100), "", ""))
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

func TestOrderBook_OCOProcessLimit(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(
		matchingo.NewStopLimitOrder("oco-2", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(150), matchingo.FromInt(101), "oco-1"),
	)
	ob.Process(matchingo.NewLimitOrder("o1", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(101), "", ""))
	ob.Process(matchingo.NewLimitOrder("o2", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(101), "", ""))

	ob.Process(matchingo.NewLimitOrder("oco-1", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), "", "oco-2"))

	done, err := ob.Process(matchingo.NewLimitOrder("simple-1", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(100), "", ""))
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

func TestOrderBook_MarketProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "", matchingo.FromInt(2))

	done, err := ob.Process(
		matchingo.NewMarketQuoteOrder("order-buy-3", matchingo.Buy, matchingo.FromInt(300)),
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trades) != 3 {
		t.Fatal("Invalid participants length")
	}

	// 200 at price 100 plus 0.909 base units at price 110 costs 299.990.
	if !done.Left.Equal(matchingo.FromIntScaled(10)) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(matchingo.FromIntScaled(299990)) {
		t.Fatal("Wrong quantity processed")
	}

	if !done.Order.IsCanceled() {
		t.Fatal("order is not canceled")
	}

	done, err = ob.Process(matchingo.NewMarketOrder("order-sell-12", matchingo.Sell, matchingo.FromInt(12)))
	if err != nil {
		t.Fatal(err)
	}

	if len(done.Trades) != 6 {
		t.Fatal("Invalid participants length")
	}

	if !done.Left.Equal(matchingo.FromInt(2)) {
		t.Fatal("Wrong quantity left")
	}

	if !done.Processed.Equal(matchingo.FromInt(10)) {
		t.Fatal("Wrong quantity processed")
	}

	if !done.Order.IsCanceled() {
		t.Fatal("order is not canceled")
	}
}

func TestOrderBook_StopOrderProcess(t *testing.T) {
	ob := matchingo.NewOrderBook()

	stop := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		matchingo.FromInt(10),

		"",
	)

	ob.Process(stop)

	if ob.Stop.Len() != 1 {
		t.Fatal("stop book is broken")
	}

	ob.Process(matchingo.NewLimitOrder("order-limit-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(10), "", ""))
	ob.Process(matchingo.NewLimitOrder("order-limit-2", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(10), "", ""))

	if ob.Stop.Len() != 0 {
		t.Fatal("stop book is broken")
	}
}

func TestOrderBook_PriceCalculation(t *testing.T) {
	ob := matchingo.NewOrderBook()
	addDepth(ob, "05-", matchingo.FromInt(10))
	addDepth(ob, "10-", matchingo.FromInt(10))
	addDepth(ob, "15-", matchingo.FromInt(10))

	price, err := ob.CalculateMarketPrice(matchingo.Buy, matchingo.FromInt(115))
	if err != nil {
		t.Fatal(err)
	}

	if !price.Equal(matchingo.FromInt(13150)) {
		t.Fatal("invalid price", price)
	}

	price, err = ob.CalculateMarketPrice(matchingo.Buy, matchingo.FromInt(200))
	if err == nil {
		t.Fatal("invalid quantity count")
	}

	if !price.Equal(matchingo.FromInt(18000)) {
		t.Fatal("invalid price", price)
	}

	// -------

	price, err = ob.CalculateMarketPrice(matchingo.Sell, matchingo.FromInt(115))
	if err != nil {
		t.Fatal(err)
	}

	if !price.Equal(matchingo.FromInt(8700)) {
		t.Fatal("invalid price", price)
	}

	price, err = ob.CalculateMarketPrice(matchingo.Sell, matchingo.FromInt(200))
	if err == nil {
		t.Fatal("invalid quantity count")
	}

	if !price.Equal(matchingo.FromInt(10500)) {
		t.Fatal("invalid price", price)
	}
}

func TestOrderBook_PartialFillVolumeConservation(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("A", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	ob.Process(matchingo.NewLimitOrder("B", matchingo.Buy, matchingo.FromInt(5), matchingo.FromInt(100), "", ""))

	depth := ob.Depth()
	if depth.Bid["100.000"] != "15.000" {
		t.Fatalf("expected bid volume 15, got %s", depth.Bid["100.000"])
	}

	ob.Process(matchingo.NewLimitOrder("S", matchingo.Sell, matchingo.FromInt(3), matchingo.FromInt(100), "", ""))

	depth = ob.Depth()
	if depth.Bid["100.000"] != "12.000" {
		t.Fatalf("expected bid volume 12 after partial fill, got %s", depth.Bid["100.000"])
	}
}

func TestOrderBook_PartialFillVolumeConservationSingleOrder(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("X", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(100), "", ""))
	ob.Process(matchingo.NewLimitOrder("S1", matchingo.Sell, matchingo.FromInt(4), matchingo.FromInt(100), "", ""))

	depth := ob.Depth()
	if depth.Bid["100.000"] != "6.000" {
		t.Fatalf("expected bid volume 6 after partial fill, got %s", depth.Bid["100.000"])
	}

	ob.Process(matchingo.NewLimitOrder("S2", matchingo.Sell, matchingo.FromInt(1), matchingo.FromInt(100), "", ""))

	depth = ob.Depth()
	if depth.Bid["100.000"] != "5.000" {
		t.Fatalf("expected bid volume 5 after second partial fill, got %s", depth.Bid["100.000"])
	}
}

var benchOb = matchingo.NewOrderBook()
var BenchPrice = matchingo.FromInt(16)
var BenchQuantity = matchingo.FromInt(150)

// benchIDs are pre-allocated order IDs to avoid string formatting overhead in benchmarks.
var benchIDs = func() []string {
	ids := make([]string, 1024)
	for i := range ids {
		ids[i] = fmt.Sprintf("buy-%d", i)
	}
	return ids
}()

func BenchmarkAppendLimitOrders(b *testing.B) {
	stopwatch := time.Now()
	for i := 0; i < b.N; i++ {
		benchOb.Process(matchingo.NewLimitOrder(benchIDs[i%len(benchIDs)], matchingo.Buy, BenchQuantity, BenchPrice, "", ""))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("Elapsed: %s\nTransactions per second (avg): %f\n", elapsed, float64(b.N*2)/elapsed.Seconds())
}

func BenchmarkLimitOrders(b *testing.B) {
	stopwatch := time.Now()
	for i := 0; i < b.N; i++ {
		benchOb.Process(matchingo.NewLimitOrder(benchIDs[i%len(benchIDs)], matchingo.Sell, BenchQuantity, BenchPrice, "", ""))
	}
	for i := 0; i < b.N; i++ {
		benchOb.Process(matchingo.NewLimitOrder(benchIDs[i%len(benchIDs)], matchingo.Buy, BenchQuantity, BenchPrice, "", ""))
	}
	elapsed := time.Since(stopwatch)
	fmt.Printf("Elapsed: %s\nTransactions per second (avg): %f\n", elapsed, float64(b.N*2)/elapsed.Seconds())
}
