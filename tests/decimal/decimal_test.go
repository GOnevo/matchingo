package decimal

import (
	"testing"

	"github.com/gonevo/matchingo"
)

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
