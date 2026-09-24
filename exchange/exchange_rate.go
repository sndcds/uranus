package exchange

import "time"

type ExchangeRate struct {
	Date     time.Time
	Currency string
	Rate     float64
}

func (r ExchangeRate) ToEUR(amount float64) float64 {
	if r.Currency == "EUR" {
		return amount
	}

	return amount / r.Rate
}
