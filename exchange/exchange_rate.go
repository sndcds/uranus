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

	reader := csv.NewReader(resp.Body)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read ECB CSV header: %w", err)
	}

	columns := make(map[string]int, len(header))

	for i, column := range header {
		columns[strings.TrimSpace(column)] = i
	}

	currencyIndex, ok := columns["CURRENCY"]
	if !ok {
		return nil, fmt.Errorf("ECB CSV has no CURRENCY column")
	}

	dateIndex, ok := columns["TIME_PERIOD"]
	if !ok {
		return nil, fmt.Errorf("ECB CSV has no TIME_PERIOD column")
	}

	rateIndex, ok := columns["OBS_VALUE"]
	if !ok {
		return nil, fmt.Errorf("ECB CSV has no OBS_VALUE column")
	}

	var rates []ExchangeRate

	for {
		record, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read ECB CSV: %w", err)
		}

		if len(record) <= maxIndex(currencyIndex, dateIndex, rateIndex) {
			return nil, fmt.Errorf("invalid ECB CSV record")
		}

		date, err := time.Parse(
			"2006-01-02",
			record[dateIndex],
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse ECB date %q: %w",
				record[dateIndex],
				err,
			)
		}

		rate, err := strconv.ParseFloat(
			record[rateIndex],
			64,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse ECB rate %q: %w",
				record[rateIndex],
				err,
			)
		}

		rates = append(rates, ExchangeRate{
			Date:     date,
			Currency: record[currencyIndex],
			Rate:     rate,
		})
	}

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
