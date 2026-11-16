package tests

import (
	"github.com/gonevo/matchingo"
	"github.com/nikolaydubina/fpdecimal"
	"testing"
)

func TestMissingRemovalPreservesSide(t *testing.T) {
	for _, side := range []*matchingo.OrderSide{matchingo.NewOrderSideAsk(), matchingo.NewOrderSideBid()} {
		live := matchingo.NewLimitOrder("live", matchingo.Buy, fpdecimal.FromInt(3), fpdecimal.FromInt(10), "", "")
		side.Append(live)
		for _, price := range []int{10, 20} {
			missing := matchingo.NewLimitOrder("missing", matchingo.Buy, fpdecimal.FromInt(1), fpdecimal.FromInt(price), "", "")
			if side.Remove(missing) != nil {
				t.Fatal("missing order reported removed")
			}
			if side.Len() != 1 || side.Depth() != 1 || !side.BestPriceQueue().Volume().Equal(fpdecimal.FromInt(3)) {
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

func TestQueueInvalidRemovalPreservesVolume(t *testing.T) {
	queue := matchingo.NewOrderQueue(fpdecimal.FromInt(10))
	live := matchingo.NewLimitOrder("live", matchingo.Buy, fpdecimal.FromInt(3), fpdecimal.FromInt(10), "", "")
	queue.Append(live)
	for _, index := range []int{-2, -1, 1, 100} {
		if queue.RemoveIndex(index) {
			t.Fatal("invalid index removed order")
		}
		if queue.Len() != 1 || !queue.Volume().Equal(fpdecimal.FromInt(3)) {
			t.Fatal("invalid removal changed queue")
		}
	}
	if queue.Remove(nil) {
		t.Fatal("nil removal reported success")
	}
	if !queue.RemoveIndex(0) || queue.Len() != 0 || !queue.Volume().Equal(fpdecimal.Zero) {
		t.Fatal("valid removal failed")
	}
}
