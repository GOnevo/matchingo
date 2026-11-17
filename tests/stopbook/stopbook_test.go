package stopbook

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestStopBook_Activate(t *testing.T) {
	stopBook := matchingo.NewStopBook()

	o1 := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
		matchingo.FromInt(20),
		"",
	)

	stopBook.Append(o1)
	stopBook.Append(o2)

	if stopBook.Len() != 2 {
		t.Fatal("invalid orders count")
	}

	slice := stopBook.Activate(matchingo.FromInt(10), matchingo.Buy)

	if len(slice) != 1 {
		t.Fatal("invalid slice count")
	}

	if stopBook.Len() != 1 {
		t.Fatal("invalid orders count")
	}

	o3 := matchingo.NewStopLimitOrder(
		"order-3",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
		matchingo.FromInt(20),
		"",
	)

	stopBook.Append(o3)

	slice = stopBook.Activate(matchingo.FromInt(20), matchingo.Buy)

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
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
		matchingo.FromInt(20),
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
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		matchingo.FromInt(10),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		matchingo.FromInt(10),
		matchingo.FromInt(20),
		matchingo.FromInt(20),
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

func TestStopBook_String(t *testing.T) {
	stopBook := matchingo.NewStopBook()
	s := stopBook.String()
	if s != "" {
		t.Fatal("empty stopbook String should be empty")
	}

	o1 := matchingo.NewStopLimitOrder("stop-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(10), matchingo.FromInt(10), "")
	stopBook.Append(o1)
	s = stopBook.String()
	if s == "" {
		t.Fatal("stopbook String should not be empty after append")
	}
}

func TestStopBook_AppendDuplicateID(t *testing.T) {
	stopBook := matchingo.NewStopBook()
	o1 := matchingo.NewStopLimitOrder("dup-1", matchingo.Buy, matchingo.FromInt(10), matchingo.FromInt(10), matchingo.FromInt(10), "")
	stopBook.Append(o1)
	stopBook.Append(o1)
	if stopBook.Len() != 1 {
		t.Fatal("duplicate append should not increase count")
	}
}
