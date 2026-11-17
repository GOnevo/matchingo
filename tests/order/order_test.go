package order

import (
	"testing"
	"time"

	"github.com/gonevo/matchingo"
)

func TestOrder_NewMarket(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewMarketOrder("id", matchingo.Buy, matchingo.FromInt(-1))
	}()
}

func TestOrder_NewStop(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(1), matchingo.FromInt(0), "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(0), matchingo.FromInt(1), "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, matchingo.FromInt(0), matchingo.FromInt(1), matchingo.FromInt(1), "")
	}()
}

func TestOrder_NewLimit(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, matchingo.FromInt(0), matchingo.FromInt(1), "", "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(0), "", "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(1), "FAKE", "")
	}()
}

func TestOrder_NewMarketQuoteOrder(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketQuoteOrder should have panic!")
			}
		}()

		matchingo.NewMarketQuoteOrder("id", matchingo.Buy, matchingo.FromInt(-1))
	}()

	order := matchingo.NewMarketQuoteOrder("quote-1", matchingo.Buy, matchingo.FromInt(10))
	if order.ID() != "quote-1" {
		t.Fatal("wrong order ID")
	}
	if order.Side() != matchingo.Buy {
		t.Fatal("wrong side")
	}
	if !order.IsQuote() {
		t.Fatal("expected isQuote to be true")
	}
	if !order.IsMarketOrder() {
		t.Fatal("expected market order type")
	}
}

func TestOrder_ActivateStopOrder(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("ActivateStopOrder should have panic on non-stop order!")
			}
		}()

		order := matchingo.NewLimitOrder("limit-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(10), "", "")
		order.ActivateStopOrder()
	}()

	order := matchingo.NewStopLimitOrder("stop-1", matchingo.Sell, matchingo.FromInt(5), matchingo.FromInt(20), matchingo.FromInt(15), "")
	if !order.IsStopOrder() {
		t.Fatal("expected stop order")
	}
	order.ActivateStopOrder()
	if order.IsStopOrder() {
		t.Fatal("order should no longer be a stop order")
	}
	if !order.IsLimitOrder() {
		t.Fatal("order should be a limit order after activation")
	}
	if !order.StopPrice().Equal(matchingo.Zero) {
		t.Fatal("stop price should be zero after activation")
	}
}

func TestOrder_String(t *testing.T) {
	order := matchingo.NewLimitOrder("str-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(100), "", "")
	s := order.String()
	if s == "" {
		t.Fatal("String() should not return empty string")
	}
	if !order.IsLimitOrder() {
		t.Fatal("order type changed")
	}
}

func TestOrder_CreatedAt(t *testing.T) {
	ob := matchingo.NewOrderBook()
	o := matchingo.NewLimitOrder("id", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(100), "", "")

	if !o.CreatedAt().IsZero() {
		t.Fatal("CreatedAt should be zero before Process")
	}

	before := time.Now()
	ob.Process(o)
	after := time.Now()

	if o.CreatedAt().IsZero() {
		t.Fatal("CreatedAt should not be zero after Process")
	}
	if o.CreatedAt().Before(before) || o.CreatedAt().After(after) {
		t.Errorf("CreatedAt %v not in [%v, %v]", o.CreatedAt(), before, after)
	}
}
