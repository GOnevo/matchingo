package decimal

import (
	"testing"

	"github.com/gonevo/matchingo"
)

func TestDecimal_DAcceptsDefinedTypes(t *testing.T) {
	type signed int32
	type unsigned uint64
	type fractional float32
	type numericText string

	got := []matchingo.Decimal{
		matchingo.D(signed(-12)),
		matchingo.D(unsigned(12)),
		matchingo.D(fractional(1.25)),
		matchingo.D(numericText("1.25")),
	}
	want := []matchingo.Decimal{
		matchingo.D(-12), matchingo.D(12), matchingo.D(1.25), matchingo.D("1.25"),
	}

	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("value %d: got %s, want %s", i, got[i], want[i])
		}
	}
}

func TestDecimal_FromFloat(t *testing.T) {
	d := matchingo.FromFloat(3.14)
	if d.String() == "" {
		t.Fatal("FromFloat should return a non-empty decimal")
	}
	d2 := matchingo.FromFloat(0.0)
	if !d2.Equal(matchingo.Zero) {
		t.Fatal("FromFloat(0.0) should equal Zero")
	}
}

func TestDecimal_FromFloat_Precision(t *testing.T) {
	d := matchingo.FromFloat(1.23456)
	if d.String() == "" {
		t.Fatal("FromFloat should return a non-empty decimal")
	}
}
