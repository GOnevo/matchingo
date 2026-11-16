package orderqueue

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonevo/matchingo"
)

func TestOrderQueue_SliceEmpty(t *testing.T) {
	price := matchingo.FromInt(100)
	oq := matchingo.NewOrderQueue(price)
	slice := oq.Slice()
	if slice != nil {
		t.Fatal("Slice on empty queue should return nil")
	}
}

func TestOrderQueue_Basic(t *testing.T) {
	price := matchingo.FromInt(100)
	oq := matchingo.NewOrderQueue(price)

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		matchingo.FromInt(100),
		matchingo.FromInt(100),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		matchingo.FromInt(100),
		matchingo.FromInt(100),
		"",
		"",
	)

	oq.Append(o1)
	oq.Append(o2)

	if oq.Orders.Len() != 2 {
		t.Fatalf("Invalid orders count(have: %d, want: 2)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(matchingo.FromInt(200)) {
		t.Fatalf("Invalid order volume (have: %s, want: 200)", oq.Volume())
	}

	o3 := matchingo.NewLimitOrder(
		"order-3",
		matchingo.Buy,
		matchingo.FromInt(200),
		matchingo.FromInt(200),
		"",
		"",
	)

	oq.Append(o3)

	if oq.Orders.Len() != 3 {
		t.Fatalf("Invalid orders count(have: %d, want: 3)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(matchingo.FromInt(400)) {
		t.Fatalf("Invalid order volume (have: %s, want: 400)", oq.Volume())
	}

	if oq.RemoveByID("order-3") != true {
		t.Fatalf("RemoveByID not work")
	}

	if oq.Orders.Len() != 2 {
		t.Fatalf("Invalid orders count(have: %d, want: 2)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(matchingo.FromInt(200)) {
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

	if !oq.Volume().Equal(matchingo.FromInt(100)) {
		t.Fatalf("Invalid order volume (have: %s, want: 100)", oq.Volume())
	}
}

func TestOrderQueue_Slice(t *testing.T) {
	price := matchingo.FromInt(100)
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

	if !oq.Volume().Equal(matchingo.FromInt(200)) {
		t.Fatalf("Invalid order volume (have: %s, want: 200)", oq.Volume())
	}

	slice := oq.Slice()

	if len(slice) != 2 {
		t.Fatalf("Invalid slice length (have: %d, want: 2)", len(slice))
	}

	if oq.Orders.Len() != 0 {
		t.Fatalf("Invalid orders count(have: %d, want: 0)", oq.Orders.Len())
	}

	if !oq.Volume().Equal(matchingo.FromInt(0)) {
		t.Fatalf("Invalid order volume (have: %s, want: 0)", oq.Volume())
	}
}

func TestOrderQueue_Update(t *testing.T) {
	price := matchingo.FromInt(100)
	oq := matchingo.NewOrderQueue(price)

	o1 := matchingo.NewLimitOrder(
		"order-1",
		matchingo.Buy,
		matchingo.FromInt(100),
		matchingo.FromInt(100),
		"",
		"",
	)

	o2 := matchingo.NewLimitOrder(
		"order-2",
		matchingo.Buy,
		matchingo.FromInt(100),
		matchingo.FromInt(100),
		"",
		"",
	)

	oq.Append(o1)
	oq.Append(o2)

	headOrder := oq.First()
	consumed := matchingo.FromInt(55)
	headOrder.DecreaseQuantity(consumed)
	oq.UpdateVolume(consumed)

	headOrder = oq.First()

	if headOrder.Quantity().String() != "45.000" {
		t.Fatalf("Invalid new price (have: %s, want: 45.000)", headOrder.Quantity().String())
	}

	if !oq.Volume().Equal(matchingo.FromInt(145)) {
		t.Fatalf("Invalid order volume (have: %s, want: 145)", oq.Volume())
	}
}

var BenchPrice = matchingo.FromInt(16)
var BenchQuantity = matchingo.FromInt(150)

var BenchOrderQueue = matchingo.NewOrderQueue(BenchPrice)

func BenchmarkOrderQueue(b *testing.B) {
	stopwatch := time.Now()

	for i := 1; i < b.N; i++ {
		BenchOrderQueue.Append(matchingo.NewLimitOrder(
			fmt.Sprintf("order-%d", i),
			matchingo.Buy,
			matchingo.FromInt(int(i)),
			BenchPrice,
			"",
			"",
		))
	}

	elapsed := time.Since(stopwatch)
	fmt.Printf("\n\nElapsed: %s\nTransactions per second: %f\n", elapsed, float64(b.N)/elapsed.Seconds())
}
