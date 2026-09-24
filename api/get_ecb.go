package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/exchange"
)

func (h *ApiHandler) ECBTest(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-ecb-test")
	ctx := gc.Request.Context()

	var data struct {
		ImportedRates int     `json:"imported_rates"`
		Rate          any     `json:"rate"`
		AmountDKK     float64 `json:"amount_dkk"`
		AmountEUR     float64 `json:"amount_eur"`
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// 1. Fetch the latest ECB reference rates.

		rates, err := exchange.FetchECBRates(ctx)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusBadGateway,
				Err:  fmt.Errorf("fetch ECB rates: %w", err),
			}
		}

		// 2. Store the rates in PostgreSQL.

		if err := exchange.StoreExchangeRates(ctx, tx, rates); err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("store ECB rates: %w", err),
			}
		}

		data.ImportedRates = len(rates)

		// 3. Example: convert a DKK price to EUR using the event date.

		eventDate := time.Date(
			2026,
			time.September,
			23,
			0,
			0,
			0,
			0,
			time.UTC,
		)

		rate, err := exchange.GetExchangeRate(
			ctx,
			tx,
			"DKK",
			eventDate,
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("get DKK exchange rate: %w", err),
			}
		}

		eur := rate.ToEUR(150)

		data.Rate = rate
		data.AmountDKK = 150
		data.AmountEUR = eur

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.Success(http.StatusOK, data)
}
