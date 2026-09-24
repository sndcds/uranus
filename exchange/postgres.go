package exchange

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// StoreExchangeRates stores exchange rates in PostgreSQL.
//
// Existing rates for the same date and currency are updated.
func StoreExchangeRates(
	ctx context.Context,
	tx pgx.Tx,
	rates []ExchangeRate,
) error {
	if len(rates) == 0 {
		return nil
	}

	const query = `
		INSERT INTO uranus.exchange_rate (
			date,
			currency,
			rate,
			source
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (date, currency)
		DO UPDATE SET
			rate = EXCLUDED.rate,
			source = EXCLUDED.source
	`

	batch := &pgx.Batch{}

	for _, rate := range rates {
		batch.Queue(
			query,
			rate.Date,
			rate.Currency,
			rate.Rate,
			"ECB",
		)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	for range rates {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf(
				"store exchange rate: %w",
				err,
			)
		}
	}

	return nil
}

// GetExchangeRate returns the latest available exchange rate for the
// specified currency on or before the given date.
//
// The rate is expressed as:
//
//	1 EUR = rate currency
//
// For example:
//
//	DKK 7.4755
//
// If date falls on a weekend or ECB holiday, the most recent available
// rate before that date is returned.
func GetExchangeRate(
	ctx context.Context,
	tx pgx.Tx,
	currency string,
	date time.Time,
) (*ExchangeRate, error) {
	const query = `
		SELECT
			date,
			currency,
			rate
		FROM uranus.exchange_rate
		WHERE currency = $1
		  AND date <= $2
		ORDER BY date DESC
		LIMIT 1
	`

	var rate ExchangeRate

	err := tx.QueryRow(
		ctx,
		query,
		currency,
		date,
	).Scan(
		&rate.Date,
		&rate.Currency,
		&rate.Rate,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get exchange rate for %s on %s: %w",
			currency,
			date.Format("2006-01-02"),
			err,
		)
	}

	return &rate, nil
}

// GetLatestExchangeRate returns the most recently stored exchange rate
// for the specified currency.
func GetLatestExchangeRate(
	ctx context.Context,
	tx pgx.Tx,
	currency string,
) (*ExchangeRate, error) {
	const query = `
		SELECT
			date,
			currency,
			rate
		FROM uranus.exchange_rate
		WHERE currency = $1
		ORDER BY date DESC
		LIMIT 1
	`

	var rate ExchangeRate

	err := tx.QueryRow(
		ctx,
		query,
		currency,
	).Scan(
		&rate.Date,
		&rate.Currency,
		&rate.Rate,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get latest exchange rate for %s: %w",
			currency,
			err,
		)
	}

	return &rate, nil
}
