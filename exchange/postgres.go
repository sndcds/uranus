package exchange

import "time"

type ExchangeRate struct {
	Date     time.Time
	Currency string
	Rate     float64
}
