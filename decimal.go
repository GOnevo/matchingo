package matchingo

import (
	"github.com/nikolaydubina/fpdecimal"
)

// FromInt returns new Decimal instance
func FromInt(num int) fpdecimal.Decimal {
	return fpdecimal.FromInt(num)
}

// FromFloat returns new Decimal instance
func FromFloat(num float64) fpdecimal.Decimal {
	return fpdecimal.FromFloat(num)
}

// SetDecimalFraction set precision
func SetDecimalFraction(precision int) {
	fpdecimal.FractionDigits = uint8(precision)
}
