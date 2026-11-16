package orderside

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestOrderSide_RemoveMissingPreservesSide(t *testing.T) {
	for _, side := range []*matchingo.OrderSide{matchingo.NewOrderSideAsk(), matchingo.NewOrderSideBid()} {
		live := matchingo.NewLimitOrder("live", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(10), "", "")
		side.Append(live)
		for _, price := range []int{10, 20} {
			missing := matchingo.NewLimitOrder("missing", matchingo.Buy, matchingo.FromInt(1), matchingo.FromInt(price), "", "")
			if side.Remove(missing) != nil {
				t.Fatal("missing order reported removed")
			}
			if side.Len() != 1 || side.Depth() != 1 || !side.BestPriceQueue().Volume().Equal(matchingo.FromInt(3)) {
				t.Fatal("missing removal mutated side")
			}
		}
		if side.Remove(nil) != nil {
			t.Fatal("nil removal reported success")
		}
		if side.Remove(live) != live || side.Len() != 0 || side.Depth() != 0 {
			t.Fatal("live removal failed")
		}
		if side.Remove(live) != nil {
			t.Fatal("duplicate removal reported success")
		}
	}
}
