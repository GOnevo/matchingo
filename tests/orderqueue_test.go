package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"

	"github.com/shopspring/decimal"
)

func TestOrderQueue(t *testing.T) {
	price := decimal.New(100, 0)
	oq := matchingo.NewOrderQueue(price)

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		decimal.New(100, 0),
		decimal.New(100, 0),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		decimal.New(100, 0),
		decimal.New(100, 0),
		"",
		"",
	)

	oq.Append(o1)
	oq.Append(o2)

	if oq.Orders.Len() != 2 {
		t.Fatalf("Invalid orders count(have: %d, want: 2)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(decimal.New(200, 0)) {
		t.Fatalf("Invalid order volume (have: %s, want: 200)", oq.Volume())
	}

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Buy,
		decimal.New(200, 0),
		decimal.New(200, 0),
		"",
		"",
	)

	oq.Append(o3)

	if oq.Orders.Len() != 3 {
		t.Fatalf("Invalid orders count(have: %d, want: 3)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(decimal.New(400, 0)) {
		t.Fatalf("Invalid order volume (have: %s, want: 400)", oq.Volume())
	}

	if oq.RemoveByID("order-3") != true {
		t.Fatalf("RemoveByID not work")
	}

	if oq.Orders.Len() != 2 {
		t.Fatalf("Invalid orders count(have: %d, want: 2)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(decimal.New(200, 0)) {
		t.Fatalf("Invalid order volume (have: %s, want: 200)", oq.Volume())
	}

	if oq.Remove(o2) != true {
		t.Fatalf("Remove not work")
	}

	if oq.Orders.Len() != 1 {
		t.Fatalf("Invalid orders count(have: %d, want: 1)", oq.Orders.Len())
	}

	order := oq.Orders.PopFront()
	if order.ID() != o1.ID() {
		t.Fatalf("Invalid order ID")
	}

	if !oq.Volume().Equal(decimal.New(100, 0)) {
		t.Fatalf("Invalid order volume (have: %s, want: 100)", oq.Volume())
	}
}

func TestOrderQueueSlice(t *testing.T) {
	price := decimal.New(100, 0)
	oq := matchingo.NewOrderQueue(price)

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		price,
		price,
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		price,
		price,
		"",
		"",
	)

	oq.Append(o1)
	oq.Append(o2)

	if oq.Orders.Len() != 2 {
		t.Fatalf("Invalid orders count(have: %d, want: 2)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(decimal.New(200, 0)) {
		t.Fatalf("Invalid order volume (have: %s, want: 200)", oq.Volume())
	}

	slice := oq.Slice()

	if len(slice) != 2 {
		t.Fatalf("Invalid slice length (have: %d, want: 2)", len(slice))
	}

	if oq.Orders.Len() != 0 {
		t.Fatalf("Invalid orders count(have: %d, want: 0)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(decimal.New(0, 0)) {
		t.Fatalf("Invalid order volume (have: %s, want: 0)", oq.Volume())
	}
}

func TestOrderQueueUpdate(t *testing.T) {
	price := decimal.New(100, 0)
	oq := matchingo.NewOrderQueue(price)

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		decimal.New(100, 0),
		decimal.New(100, 0),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		decimal.New(100, 0),
		decimal.New(100, 0),
		"",
		"",
	)

	oq.Append(o1)
	oq.Append(o2)

	headOrder := oq.First()
	oq.DecreaseQuantity(headOrder, decimal.New(55, 0))

	headOrder = oq.First()

	if headOrder.Quantity().String() != "45" {
		t.Fatalf("Invalid new price (have: %s, want: 45)", headOrder.Quantity().String())
	}
}

func BenchmarkOrderQueue(b *testing.B) {
	price := decimal.New(100, 0)
	orderQueue := matchingo.NewOrderQueue(price)
	stopwatch := time.Now()

	for i := 0; i < b.N; i++ {
		orderQueue.Append(matchingo.NewLimitOrder(
			fmt.Sprintf("order-%d", i),
			matchingo.Buy,
			price,
			decimal.NewFromInt(int64(i)),
			"",
			"",
		))
	}

	elapsed := time.Since(stopwatch)
	fmt.Printf("\n\nElapsed: %s\nTransactions per second: %f\n", elapsed, float64(b.N)/elapsed.Seconds())
}
