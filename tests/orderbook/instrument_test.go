package orderbook

import (
	"errors"
	"testing"

	"github.com/gonevo/matchingo"
)

func instBook() *matchingo.OrderBook {
	return matchingo.NewOrderBook(matchingo.WithInstrument(matchingo.Instrument{
		Symbol:      "BTC-USD",
		TickSize:    d("0.5"),
		LotSize:     d("0.1"),
		MinQuantity: d("0.2"),
		MaxQuantity: d("10"),
		MinPrice:    d("50"),
		MaxPrice:    d("200"),
	}))
}

func lim(id string, qty, price string) *matchingo.Order {
	return matchingo.NewLimitOrder(id, matchingo.Buy, d(qty), d(price), matchingo.GTC, "")
}

func TestInstrument_RejectsViolations(t *testing.T) {
	b := instBook()
	cases := []struct {
		name string
		o    *matchingo.Order
		want error
	}{
		{"tick", lim("a", "1", "100.2"), matchingo.ErrTickSize},
		{"lot", lim("b", "1.05", "100"), matchingo.ErrLotSize},
		{"min qty", lim("c", "0.1", "100"), matchingo.ErrQuantityRange},
		{"max qty", lim("d", "10.1", "100"), matchingo.ErrQuantityRange},
		{"price below band", lim("e", "1", "49.5"), matchingo.ErrPriceBand},
		{"price above band", lim("f", "1", "200.5"), matchingo.ErrPriceBand},
		{"stop tick", matchingo.NewStopLimitOrder("g", matchingo.Buy, d("1"), d("100"), d("100.2"), ""), matchingo.ErrTickSize},
		{"stop-market band", matchingo.NewStopMarketOrder("h", matchingo.Buy, d("1"), d("300"), ""), matchingo.ErrPriceBand},
		{"market lot", matchingo.NewMarketOrder("i", matchingo.Buy, d("1.05")), matchingo.ErrLotSize},
		{"iceberg hidden lot", matchingo.NewIcebergOrder("j", matchingo.Buy, d("1.05"), d("0.5"), d("100"), matchingo.GTC, ""), matchingo.ErrLotSize},
	}
	for _, c := range cases {
		if _, err := b.Process(c.o); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
	if b.Sequence() != 0 {
		t.Fatal("rejected orders must not advance the sequence")
	}
}

func TestInstrument_AcceptsBoundaryAndQuoteOrders(t *testing.T) {
	b := instBook()
	for _, o := range []*matchingo.Order{
		lim("lo", "0.2", "50"),
		lim("hi", "10", "200"),
		lim("mid", "1.3", "100.5"),
		matchingo.NewStopMarketOrder("s", matchingo.Buy, d("1"), d("120"), ""),
	} {
		if _, err := b.Process(o); err != nil {
			t.Fatalf("%s: %v", o.ID(), err)
		}
	}
	// a quote budget is not a lot count
	if _, err := b.Process(matchingo.NewMarketQuoteOrder("q", matchingo.Sell, d("33.33"))); err != nil {
		t.Fatalf("quote order: %v", err)
	}
	if b.Instrument().Symbol != "BTC-USD" {
		t.Fatal("Instrument() should return the configuration")
	}
}

func TestInstrument_AmendmentsAndReplaceAreChecked(t *testing.T) {
	b := instBook()
	b.Process(lim("a", "1", "100"))
	b.Process(matchingo.NewStopLimitOrder("s", matchingo.Buy, d("1"), d("101"), d("100"), ""))

	if _, err := b.Modify("a", d("100.2"), d("1")); !errors.Is(err, matchingo.ErrTickSize) {
		t.Fatalf("Modify tick: %v", err)
	}
	if _, err := b.Modify("a", d("100"), d("0.05")); !errors.Is(err, matchingo.ErrLotSize) {
		t.Fatalf("Modify lot: %v", err)
	}
	if err := b.ModifyStop("s", d("100"), d("101"), d("11")); !errors.Is(err, matchingo.ErrQuantityRange) {
		t.Fatalf("ModifyStop qty: %v", err)
	}
	if _, err := b.Replace("a", lim("a", "1", "300")); !errors.Is(err, matchingo.ErrPriceBand) || b.GetOrder("a") == nil {
		t.Fatalf("Replace must fail and keep the old order: %v", err)
	}
}

func TestInstrument_InvalidDefinitionPanics(t *testing.T) {
	for name, in := range map[string]matchingo.Instrument{
		"negative": {TickSize: d("-1")},
		"min>max":  {MinQuantity: d("5"), MaxQuantity: d("1")},
		"band":     {MinPrice: d("5"), MaxPrice: d("1")},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("%s: expected panic", name)
				}
			}()
			matchingo.WithInstrument(in)
		}()
	}
}
