package tests

import (
	"github.com/gonevo/matchingo"
	"github.com/shopspring/decimal"
	"testing"
)

func TestTrade_NewTrade(t *testing.T) {

	main := matchingo.NewLimitOrder(
		"main-1",
		matchingo.Buy,
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
		"",
	)

	trade := matchingo.NewTrade(main)

	trade.Append(matchingo.NewLimitOrder(
		"part-1",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
		"",
	), decimal.New(10, 0), decimal.New(10, 0))

	trade.Append(matchingo.NewLimitOrder(
		"part-2",
		matchingo.Sell,
		decimal.New(10, 0),
		decimal.New(10, 0),
		"",
		"",
	), decimal.New(10, 0), decimal.New(10, 0))
}
