package tests

import (
	"testing"

	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
)

func TestStopBook_Activate(t *testing.T) {
	stopBook := matchingo.NewStopBook()

	o1 := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
		fpdecimal.FromInt(20),
		"",
	)

	stopBook.Append(o1)
	stopBook.Append(o2)

	if stopBook.Len() != 2 {
		t.Fatal("invalid orders count")
	}

	slice := stopBook.Activate(fpdecimal.FromInt(10))

	if len(slice) != 1 {
		t.Fatal("invalid slice count")
	}

	if stopBook.Len() != 1 {
		t.Fatal("invalid orders count")
	}

	o3 := matchingo.NewStopLimitOrder(
		"order-3",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
		fpdecimal.FromInt(20),
		"",
	)

	stopBook.Append(o3)

	slice = stopBook.Activate(fpdecimal.FromInt(20))

	if len(slice) != 2 {
		t.Fatal("invalid slice count")
	}

	if stopBook.Len() != 0 {
		t.Fatal("invalid orders count")
	}
}

func TestStopBook_Remove(t *testing.T) {
	stopBook := matchingo.NewStopBook()

	o1 := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
		fpdecimal.FromInt(20),
		"",
	)

	stopBook.Append(o1)
	stopBook.Append(o2)

	if stopBook.Len() != 2 {
		t.Fatal("invalid orders count")
	}

	stopBook.Remove(o1)

	if stopBook.Len() != 1 {
		t.Fatal("invalid orders count")
	}

}

func TestStopBook_RemoveByID(t *testing.T) {
	stopBook := matchingo.NewStopBook()

	o1 := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(10),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		fpdecimal.FromInt(10),
		fpdecimal.FromInt(20),
		fpdecimal.FromInt(20),
		"",
	)

	stopBook.Append(o1)
	stopBook.Append(o2)

	if stopBook.Len() != 2 {
		t.Fatal("invalid orders count")
	}

	stopBook.RemoveByID("order-1")

	if stopBook.Len() != 1 {
		t.Fatal("invalid orders count")
	}

	stopBook.RemoveByID("order-3")

	stopBook.RemoveByID("order-2")

	if stopBook.Len() != 0 {
		t.Fatal("invalid orders count")
	}
}
