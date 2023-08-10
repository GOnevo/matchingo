package tests

import (
	"testing"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func TestCancelOrder(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("order-1", matchingo.Sell, fpdecimal.FromInt(10), fpdecimal.FromInt(10), "", ""))
	ob.Process(matchingo.NewStopLimitOrder("order-2", matchingo.Sell, fpdecimal.FromInt(10), fpdecimal.FromInt(10), fpdecimal.FromInt(11), ""))

	if ob.CancelOrder("order-2").IsStopOrder() != true {
		t.Fatal("canceling stop order not work")
	}

	if ob.CancelOrder("order-2") != nil {
		t.Fatal("canceling stop order not work")
	}

	if ob.CancelOrder("order-1").IsLimitOrder() != true {
		t.Fatal("canceling stop order not work")
	}
	if ob.CancelOrder("order-1") != nil {
		t.Fatal("canceling stop order not work")
	}
}

func TestCancelActivatedOrder(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewStopLimitOrder("order-1", matchingo.Sell, fpdecimal.FromInt(10), fpdecimal.FromInt(10), fpdecimal.FromInt(11), ""))
	ob.Process(matchingo.NewLimitOrder("order-2", matchingo.Sell, fpdecimal.FromInt(10), fpdecimal.FromInt(11), "", ""))
	ob.Process(matchingo.NewLimitOrder("order-3", matchingo.Buy, fpdecimal.FromInt(10), fpdecimal.FromInt(11), "", ""))

	if ob.CancelOrder("order-1").IsLimitOrder() != true {
		t.Fatal("canceling stop order not work")
	}

	if ob.CancelOrder("order-1") != nil {
		t.Fatal("canceling stop order not work")
	}
}
