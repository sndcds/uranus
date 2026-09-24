package exchange

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const ecbRatesURL = "https://data-api.ecb.europa.eu/service/data/EXR/D..EUR.SP00.A?lastNObservations=1&format=csvdata"

// FetchECBRates downloads the latest available ECB reference exchange
// rates and returns them as ExchangeRate values.
//
// Rates are expressed as:
//
//	1 EUR = rate currency
//
// For example:
//
//	DKK 7.4755
//
// means that 1 EUR equals 7.4755 DKK.
func FetchECBRates(ctx context.Context) ([]ExchangeRate, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		ecbRatesURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create ECB request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch ECB rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"ECB returned HTTP status %d",
			resp.StatusCode,
		)
	}

	rates, err := parseECBRates(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse ECB rates: %w", err)
	}

	return rates, nil
}

// parseECBRates parses ECB SDMX-CSV data.
//
// This function is intentionally independent of HTTP so that the CSV
// parsing can be tested without accessing the ECB.
//
// If the CSV contains multiple observation dates, only the latest
// date is returned.
func parseECBRates(r io.Reader) ([]ExchangeRate, error) {
	reader := csv.NewReader(r)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	columns := make(map[string]int, len(header))

	for i, column := range header {
		columns[strings.TrimSpace(column)] = i
	}

	currencyIndex, ok := columns["CURRENCY"]
	if !ok {
		return nil, fmt.Errorf("CSV has no CURRENCY column")
	}

	dateIndex, ok := columns["TIME_PERIOD"]
	if !ok {
		return nil, fmt.Errorf("CSV has no TIME_PERIOD column")
	}

	rateIndex, ok := columns["OBS_VALUE"]
	if !ok {
		return nil, fmt.Errorf("CSV has no OBS_VALUE column")
	}

	requiredIndex := maxIndex(
		currencyIndex,
		dateIndex,
		rateIndex,
	)

	var rates []ExchangeRate
	var latestDate time.Time

	for {
		record, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read CSV record: %w", err)
		}

		if len(record) <= requiredIndex {
			return nil, fmt.Errorf(
				"CSV record has %d columns, expected at least %d",
				len(record),
				requiredIndex+1,
			)
		}

		currency := strings.TrimSpace(record[currencyIndex])

		if currency == "" {
			return nil, fmt.Errorf("CSV record has empty currency")
		}

		date, err := time.Parse(
			"2006-01-02",
			strings.TrimSpace(record[dateIndex]),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse date %q: %w",
				record[dateIndex],
				err,
			)
		}

		rate, err := strconv.ParseFloat(
			strings.TrimSpace(record[rateIndex]),
			64,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse rate %q for %s: %w",
				record[rateIndex],
				currency,
				err,
			)
		}

		if rate <= 0 {
			return nil, fmt.Errorf(
				"invalid rate %q for %s",
				record[rateIndex],
				currency,
			)
		}

		// A newer date was found. Discard all previously collected
		// rates because only the latest observation date is relevant.
		if latestDate.IsZero() || date.After(latestDate) {
			latestDate = date
			rates = rates[:0]
		}

		// Ignore records from older dates.
		if !date.Equal(latestDate) {
			continue
		}

		rates = append(rates, ExchangeRate{
			Date:     date,
			Currency: currency,
			Rate:     rate,
		})
	}

	if len(rates) == 0 {
		return nil, fmt.Errorf("ECB CSV contains no exchange rates")
	}

	// EUR is the base currency of the ECB reference rates.
	rates = append(rates, ExchangeRate{
		Date:     latestDate,
		Currency: "EUR",
		Rate:     1,
	})

	return rates, nil
}

func maxIndex(values ...int) int {
	max := values[0]

	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}

	return max
}
