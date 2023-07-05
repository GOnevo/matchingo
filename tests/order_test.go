package tests

import (
	"testing"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func TestOrder_Market(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewMarketOrder("id", matchingo.Buy, fpdecimal.FromInt(-1))
	}()
}

func TestOrder_Stop(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(1), fpdecimal.FromInt(0), "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(0), fpdecimal.FromInt(1), "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewMarketOrder should have panic!")
			}
		}()

		matchingo.NewStopLimitOrder("id", matchingo.Buy, fpdecimal.FromInt(0), fpdecimal.FromInt(1), fpdecimal.FromInt(1), "")
	}()
}

func TestOrder_Limit(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, fpdecimal.FromInt(0), fpdecimal.FromInt(1), "", "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(0), "", "")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("NewLimitOrder should have panic!")
			}
		}()

		matchingo.NewLimitOrder("id", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(1), "FAKE", "")
	}()
}
