package order

import (
	"testing"

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
