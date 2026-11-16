package orderqueue

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestOrderQueue_InvalidRemovalPreservesVolume(t *testing.T) {
	queue := matchingo.NewOrderQueue(matchingo.FromInt(10))
	live := matchingo.NewLimitOrder("live", matchingo.Buy, matchingo.FromInt(3), matchingo.FromInt(10), "", "")
	queue.Append(live)
	for _, index := range []int{-2, -1, 1, 100} {
		if queue.RemoveIndex(index) {
			t.Fatal("invalid index removed order")
		}
		if queue.Len() != 1 || !queue.Volume().Equal(matchingo.FromInt(3)) {
			t.Fatal("invalid removal changed queue")
		}
	}
	if queue.Remove(nil) {
		t.Fatal("nil removal reported success")
	}
	if !queue.RemoveIndex(0) || queue.Len() != 0 || !queue.Volume().Equal(matchingo.Zero) {
		t.Fatal("valid removal failed")
	}
}
