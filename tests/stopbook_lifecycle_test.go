package tests

import (
	"testing"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func TestStopBookReleasesIDsAfterRemovalAndActivation(t *testing.T) {
	for _, operation := range []string{"remove", "activate"} {
		t.Run(operation, func(t *testing.T) {
			book := matchingo.NewStopBook()
			original := matchingo.NewStopLimitOrder("stop", matchingo.Buy, fpdecimal.FromInt(2), fpdecimal.FromInt(9), fpdecimal.FromInt(10), "")
			book.Append(original)

			if operation == "remove" {
				if book.Remove(original) != original {
					t.Fatal("Remove did not return the original order")
				}
			} else if orders := book.Activate(fpdecimal.FromInt(10)); len(orders) != 1 || orders[0].ID() != original.ID() {
				t.Fatalf("unexpected activated orders: %v", orders)
			}
			if book.Len() != 0 {
				t.Fatalf("remaining stop count = %d, want 0", book.Len())
			}

			replacement := matchingo.NewStopLimitOrder("stop", matchingo.Buy, fpdecimal.FromInt(3), fpdecimal.FromInt(8), fpdecimal.FromInt(11), "")
			book.Append(replacement)
			if book.Len() != 1 {
				t.Fatalf("replacement was not stored: count = %d", book.Len())
			}
			if removed := book.RemoveByID("stop"); removed != replacement {
				t.Fatalf("RemoveByID returned %v, want replacement", removed)
			}
			if book.Len() != 0 || book.RemoveByID("stop") != nil {
				t.Fatal("replacement was not completely removed")
			}
		})
	}
}

func TestStopBookCannotRemoveReleasedIDTwice(t *testing.T) {
	for _, operation := range []string{"remove", "activate"} {
		t.Run(operation, func(t *testing.T) {
			book := matchingo.NewStopBook()
			order := matchingo.NewStopLimitOrder("stop", matchingo.Sell, fpdecimal.FromInt(2), fpdecimal.FromInt(12), fpdecimal.FromInt(11), "")
			book.Append(order)
			if operation == "remove" {
				book.Remove(order)
			} else {
				book.Activate(fpdecimal.FromInt(11))
			}
			if removed := book.RemoveByID("stop"); removed != nil {
				t.Fatalf("removed a stale order: %v", removed)
			}
			if book.Len() != 0 {
				t.Fatalf("stop count = %d, want 0", book.Len())
			}
		})
	}
}

func TestCancelAndResubmitStopOrder(t *testing.T) {
	book := matchingo.NewOrderBook()
	original := matchingo.NewStopLimitOrder("stop", matchingo.Buy, fpdecimal.FromInt(2), fpdecimal.FromInt(9), fpdecimal.FromInt(10), "")
	if _, err := book.Process(original); err != nil {
		t.Fatal(err)
	}
	if book.CancelOrder("stop") != original {
		t.Fatal("original stop was not canceled")
	}
	replacement := matchingo.NewStopLimitOrder("stop", matchingo.Buy, fpdecimal.FromInt(3), fpdecimal.FromInt(8), fpdecimal.FromInt(11), "")
	if _, err := book.Process(replacement); err != nil {
		t.Fatal(err)
	}
	if book.Stop.Len() != 1 || book.GetOrder("stop") != replacement {
		t.Fatal("replacement stop is not present in both indexes")
	}
	if book.CancelOrder("stop") != replacement || book.Stop.Len() != 0 {
		t.Fatal("replacement stop could not be canceled")
	}
}
