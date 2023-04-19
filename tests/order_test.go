package tests

import (
	"github.com/gonevo/matchingo"
	"github.com/shopspring/decimal"
	"testing"
)

func TestOrder_Market(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewMarketOrder("id", matchingo.Buy, decimal.New(-1, 0))
	}()
}

func TestOrder_Stop(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, decimal.New(1, 0), decimal.New(1, 0), decimal.New(0, 0), "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, decimal.New(1, 0), decimal.New(0, 0), decimal.New(1, 0), "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, decimal.New(0, 0), decimal.New(1, 0), decimal.New(1, 0), "")
	}()
}

func TestOrder_Limit(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, decimal.New(0, 0), decimal.New(1, 0), "", "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, decimal.New(1, 0), decimal.New(0, 0), "", "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, decimal.New(1, 0), decimal.New(1, 0), "FAKE", "")
	}()
}
