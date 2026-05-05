package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) UpdateVenueFields(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-venue-fields")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	venueUuid := gc.Param("venueUuid")
	if venueUuid == "" {
		apiRequest.Required("venueUuid is required")
		return
	}
	apiRequest.SetMeta("venue_uuid", venueUuid)

	var payload struct {
		Name         NullableField[string]  `json:"name"`
		Type         NullableField[string]  `json:"type"`
		Description  NullableField[string]  `json:"description"`
		ContactEmail NullableField[string]  `json:"contact_email"`
		ContactPhone NullableField[string]  `json:"contact_phone"`
		WebLink      NullableField[string]  `json:"web_link"`
		Street       NullableField[string]  `json:"street"`
		HouseNumber  NullableField[string]  `json:"house_number"`
		PostalCode   NullableField[string]  `json:"postal_code"`
		City         NullableField[string]  `json:"city"`
		State        NullableField[string]  `json:"state"`
		Country      NullableField[string]  `json:"country"`
		Lon          NullableField[float64] `json:"lon"`
		Lat          NullableField[float64] `json:"lat"`
		OpenedAt     NullableField[string]  `json:"opened_at"`
		ClosedAt     NullableField[string]  `json:"closed_at"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		debugf(err.Error())
		apiRequest.PayloadError()
		return
	}

	_, ok, err := ParseNullableDateString(payload.OpenedAt, "opened_at", "2026-01-01")
	if !ok && err != nil {
		debugf(err.Error())
		apiRequest.SuccessNoData(http.StatusBadRequest, err.Error())
		return
	}

	_, ok, err = ParseNullableDateString(payload.ClosedAt, "closed_at", "2026-01-01")
	if !ok && err != nil {
		debugf(err.Error())
		apiRequest.SuccessNoData(http.StatusBadRequest, err.Error())
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	argPos = addUpdateClauseUuid7("modified_by", userUuid, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("name", payload.Name, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("type", payload.Type, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("description", payload.Description, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("contact_email", payload.ContactEmail, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("contact_phone", payload.ContactPhone, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("web_link", payload.WebLink, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("street", payload.Street, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("house_number", payload.HouseNumber, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("postal_code", payload.PostalCode, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("city", payload.City, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("state", payload.State, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("country", payload.Country, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("opened_at", payload.OpenedAt, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("closed_at", payload.ClosedAt, &setClauses, &args, argPos)

	if payload.Lon.Set && payload.Lon.Value != nil && payload.Lat.Set && payload.Lat.Value != nil {
		// Construct PostGIS POINT in WKT format
		setClauses = append(setClauses, fmt.Sprintf("point = ST_SetSRID(ST_MakePoint($%d, $%d), 4326)", argPos, argPos+1))
		args = append(args, *payload.Lon.Value, *payload.Lat.Value)
		argPos += 2
	}

	if len(setClauses) == 0 {
		apiRequest.SuccessNoData(http.StatusOK, "no fields updated")
		return
	}
	apiRequest.SetMeta("field_count", len(setClauses)-1) // subtract 1 for userUuid

	query := fmt.Sprintf(`UPDATE %s.venue SET %s WHERE uuid = $%d::uuid`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)

	args = append(args, venueUuid) // eventUuid is the last parameter

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		res, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return TxInternalError(nil)
		}

		if res.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  errors.New("event not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "venue", []string{venueUuid})
		if err != nil {
			return TxInternalError(nil)
		}

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "venue fields updated")
}
