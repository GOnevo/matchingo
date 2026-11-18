package orderbook

import (
	"strconv"
	"testing"

	"github.com/gonevo/matchingo"
)

func uniqueIDs(prefix string, n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = prefix + strconv.Itoa(i)
	}
	return ids
}

// BenchmarkEngineRest measures accepting limit orders that rest on the book (unique IDs, no match).
func BenchmarkEngineRest(b *testing.B) {
	ids := uniqueIDs("r-", b.N)
	ob := matchingo.NewOrderBook()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.Process(matchingo.NewLimitOrder(ids[i], matchingo.Buy, BenchQuantity, BenchPrice, "", ""))
	}
}

// BenchmarkEngineMatch measures a full rest + cross cycle per iteration: one resting sell, one buy that fully fills it.
func BenchmarkEngineMatch(b *testing.B) {
	sells := uniqueIDs("s-", b.N)
	buys := uniqueIDs("b-", b.N)
	ob := matchingo.NewOrderBook()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob.Process(matchingo.NewLimitOrder(sells[i], matchingo.Sell, BenchQuantity, BenchPrice, "", ""))
		ob.Process(matchingo.NewLimitOrder(buys[i], matchingo.Buy, BenchQuantity, BenchPrice, "", ""))
	}
}
