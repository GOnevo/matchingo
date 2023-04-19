package tests

import (
	"testing"

	"github.com/gonevo/matchingo"
	"github.com/shopspring/decimal"
)

func TestStopBook_Activate(t *testing.T) {
	stopBook := matchingo.NewStopBook()

	o1 := matchingo.NewStopLimitOrder(
		"order-1",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(20, 0),
		decimal.New(20, 0),
		"",
	)

	stopBook.Append(o1)
	stopBook.Append(o2)

	if stopBook.Len() != 2 {
		t.Fatal("invalid orders count")
	}

	slice := stopBook.Activate(decimal.New(10, 0))

	if len(slice) != 1 {
		t.Fatal("invalid slice count")
	}

	if stopBook.Len() != 1 {
		t.Fatal("invalid orders count")
	}

	o3 := matchingo.NewStopLimitOrder(
		"order-3",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(20, 0),
		decimal.New(20, 0),
		"",
	)

	stopBook.Append(o3)

	slice = stopBook.Activate(decimal.New(20, 0))

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
		decimal.New(10, 0),
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(20, 0),
		decimal.New(20, 0),
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
		decimal.New(10, 0),
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
	)

	o2 := matchingo.NewStopLimitOrder(
		"order-2",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(20, 0),
		decimal.New(20, 0),
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
