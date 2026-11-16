package tests

import (
	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
	"io"
	"os"
	"testing"
)

func TestProcessRejectsInvalidInputWithoutChangingBook(t *testing.T) {
	factories := map[string]func() *matchingo.Order{
		"nil":          func() *matchingo.Order { return nil },
		"unknown type": func() *matchingo.Order { return &matchingo.Order{} },
		"invalid side": func() *matchingo.Order {
			return matchingo.NewMarketOrder("bad", matchingo.Side(99), fpdecimal.FromInt(1))
		},
		"zero remaining": func() *matchingo.Order {
			o := matchingo.NewLimitOrder("bad", matchingo.Sell, fpdecimal.FromInt(1), fpdecimal.FromInt(10), "", "")
			o.SetQuantity(fpdecimal.Zero)
			return o
		},
		"canceled": func() *matchingo.Order {
			o := matchingo.NewMarketOrder("bad", matchingo.Sell, fpdecimal.FromInt(1))
			o.Cancel()
			return o
		},
		"duplicate market": func() *matchingo.Order { return matchingo.NewMarketOrder("live", matchingo.Sell, fpdecimal.FromInt(1)) },
		"duplicate stop": func() *matchingo.Order {
			return matchingo.NewStopLimitOrder("live", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(8), fpdecimal.FromInt(9), "")
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			book := matchingo.NewOrderBook()
			live := matchingo.NewLimitOrder("live", matchingo.Buy, fpdecimal.FromInt(2), fpdecimal.FromInt(10), "", "")
			if _, err := book.Process(live); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("Process panicked: %v", p)
				}
			}()
			done, err := book.Process(factory())
			if err == nil || done != nil {
				t.Errorf("expected rejection, got done=%v, err=%v", done, err)
			}
			if book.GetOrder("live") != live || !live.Quantity().Equal(fpdecimal.FromInt(2)) || book.Stop.Len() != 0 {
				t.Error("rejected input changed the book")
			}
		})
	}
}

func TestQuoteBudgetPreservesUnspentRemainder(t *testing.T) {
	for _, tc := range []struct {
		name                string
		budget, price, left string
	}{
		{"rounding remainder", "10", "3", "0.001"},
		{"below one base unit", "0.001", "2", "0.001"},
		{"exact fill", "10", "2", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parse := func(s string) fpdecimal.Decimal {
				d, err := fpdecimal.FromString(s)
				if err != nil {
					t.Fatal(err)
				}
				return d
			}
			book := matchingo.NewOrderBook()
			maker := matchingo.NewLimitOrder("maker", matchingo.Sell, fpdecimal.FromInt(10), parse(tc.price), "", "")
			if _, err := book.Process(maker); err != nil {
				t.Fatal(err)
			}
			done, err := book.Process(matchingo.NewMarketQuoteOrder("buyer", matchingo.Buy, parse(tc.budget)))
			if err != nil {
				t.Fatal(err)
			}
			if !done.Left.Equal(parse(tc.left)) {
				t.Fatalf("left=%s, want %s", done.Left, tc.left)
			}
			spent := fpdecimal.Zero
			for _, trade := range done.Trades {
				if trade.Role == matchingo.MAKER {
					spent = spent.Add(trade.Price.Mul(trade.Quantity))
				}
			}
			if !spent.Equal(done.Processed) || !spent.Add(done.Left).Equal(parse(tc.budget)) {
				t.Fatal("budget does not equal executed cost plus remainder")
			}
			if done.Left.GreaterThan(fpdecimal.Zero) && (!done.Order.IsCanceled() || len(done.Canceled) != 1) {
				t.Fatal("unspendable remainder was not canceled")
			}
		})
	}
}

func TestUnfilledMarketOrderIsReportedAsCanceled(t *testing.T) {
	book := matchingo.NewOrderBook()
	done, err := book.Process(matchingo.NewMarketOrder("empty", matchingo.Sell, fpdecimal.FromInt(2)))
	if err != nil {
		t.Fatal(err)
	}
	if !done.Left.Equal(fpdecimal.FromInt(2)) || !done.Processed.Equal(fpdecimal.Zero) || !done.Order.IsCanceled() || len(done.Canceled) != 1 {
		t.Fatal("incorrect unfilled order report")
	}
}

func TestDepthDoesNotWriteToStdout(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = write
	defer func() { os.Stdout = original; read.Close(); write.Close() }()
	matchingo.NewOrderBook().Depth()
	write.Close()
	os.Stdout = original
	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) > 0 {
		t.Fatalf("Depth leaked output: %q", output)
	}
}

func BenchmarkQueueSliceReview(b *testing.B) {
	price := fpdecimal.FromInt(10)
	order := matchingo.NewLimitOrder("benchmark", matchingo.Buy, fpdecimal.FromInt(1), price, "", "")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		queue := matchingo.NewOrderQueue(price)
		for n := 0; n < 1000; n++ {
			queue.Append(order)
		}
		if len(queue.Slice()) != 1000 {
			b.Fatal("unexpected length")
		}
	}
}
