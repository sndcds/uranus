package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminRefreshEventProjections(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-refresh-event-projection")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "missing eventUuid")
		return
	}

	code := gc.Query("code")
	debugf("code %s", code)

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Optional: authorization check
		_ = userUuid
		err := RefreshEventProjections(
			ctx,
			tx,
			"event",
			[]string{eventUuid},
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}
		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "projection refreshed")
}
