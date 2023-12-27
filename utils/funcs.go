package utils

import (
	"fmt"

	"github.com/shopspring/decimal"
)

func FloatToDecimal(value float64) decimal.Decimal {
	d, _ := decimal.NewFromString(fmt.Sprintf("%f", value))
	return d
}
