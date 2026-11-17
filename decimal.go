package matchingo

import (
	"fmt"

	"github.com/nikolaydubina/fpdecimal"
)

// Decimal is a wrapper around fpdecimal.Decimal for use throughout the matchingo package.
type Decimal struct {
	fpdecimal.Decimal
}

// Zero is the zero value for Decimal.
var Zero = Decimal{fpdecimal.Zero}

// FromInt returns new Decimal instance
func FromInt(num int) Decimal {
	return Decimal{fpdecimal.FromInt(num)}
}

// FromFloat returns new Decimal instance
func FromFloat(num float64) Decimal {
	return Decimal{fpdecimal.FromFloat(num)}
}

// LessThanOrEqual returns true if d <= other
func (d Decimal) LessThanOrEqual(other Decimal) bool {
	return d.Decimal.LessThanOrEqual(other.Decimal)
}

// GreaterThan returns true if d > other
func (d Decimal) GreaterThan(other Decimal) bool {
	return d.Decimal.GreaterThan(other.Decimal)
}

// GreaterThanOrEqual returns true if d >= other
func (d Decimal) GreaterThanOrEqual(other Decimal) bool {
	return d.Decimal.GreaterThanOrEqual(other.Decimal)
}

// LessThan returns true if d < other
func (d Decimal) LessThan(other Decimal) bool {
	return d.Decimal.LessThan(other.Decimal)
}

// Equal returns true if d == other
func (d Decimal) Equal(other Decimal) bool {
	return d.Decimal.Equal(other.Decimal)
}

// Add returns d + other
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{d.Decimal.Add(other.Decimal)}
}

// Sub returns d - other
func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{d.Decimal.Sub(other.Decimal)}
}

// Mul returns d * other
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{d.Decimal.Mul(other.Decimal)}
}

// Div returns d / other
func (d Decimal) Div(other Decimal) Decimal {
	return Decimal{d.Decimal.Div(other.Decimal)}
}

// String returns the string representation of d with trailing zeros.
func (d Decimal) String() string {
	v := d.Decimal.Scaled()
	if v == 0 {
		return fmt.Sprintf("0.%0*d", int(fpdecimal.FractionDigits), 0)
	}

	neg := v < 0
	if neg {
		v = -v
	}

	frac := int64(1)
	for i := uint8(0); i < fpdecimal.FractionDigits; i++ {
		frac *= 10
	}

	whole := v / frac
	fracPart := v % frac

	s := fmt.Sprintf("%d.%0*d", whole, int(fpdecimal.FractionDigits), fracPart)
	if neg {
		s = "-" + s
	}
	return s
}

// Compare compares d with other and returns -1, 0, or 1
func (d Decimal) Compare(other Decimal) int {
	return d.Decimal.Compare(other.Decimal)
}

// FromString returns new Decimal instance from string representation
func FromString(s string) (Decimal, error) {
	d, err := fpdecimal.FromString(s)
	return Decimal{d}, err
}

// FromIntScaled returns new Decimal instance from scaled integer
func FromIntScaled(num int64) Decimal {
	return Decimal{fpdecimal.FromIntScaled(num)}
}
