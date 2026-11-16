package orderbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestOrderBook_CancelOrder(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewLimitOrder("order-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(10), "", ""))
	ob.Process(matchingo.NewStopLimitOrder("order-2", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(10), matchingo.FromInt(11), ""))

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

func TestOrderBook_CancelActivatedOrder(t *testing.T) {
	ob := matchingo.NewOrderBook()

	ob.Process(matchingo.NewStopLimitOrder("order-1", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(10), matchingo.FromInt(11), ""))
	ob.Process(matchingo.NewLimitOrder("order-2", matchingo.Sell, matchingo.FromInt(10), matchingo.FromInt(11), "", ""))
	ob.Process(matchingo.NewLimitOrder("order-3", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(11), "", ""))

	if ob.CancelOrder("order-1").IsLimitOrder() != true {
		t.Fatal("canceling stop order not work")
	}

	if ob.CancelOrder("order-1") != nil {
		t.Fatal("canceling stop order not work")
	}
}
