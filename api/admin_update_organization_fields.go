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

func (h *ApiHandler) UpdateOrganizationFields(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-organization-fields")
	ctx := gc.Request.Context()

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "orgUuid is required")
		return
	}
	apiRequest.SetMeta("org_uuid", orgUuid)

	var payload struct {
		Name            NullableField[string]  `json:"name"`
		Description     NullableField[string]  `json:"description"`
		LegalForm       NullableField[string]  `json:"legal_form"`
		ContactEmail    NullableField[string]  `json:"contact_email"`
		ContactPhone    NullableField[string]  `json:"contact_phone"`
		WebLink         NullableField[string]  `json:"web_link"`
		Street          NullableField[string]  `json:"street"`
		HouseNumber     NullableField[string]  `json:"house_number"`
		AddressAddition NullableField[string]  `json:"address_addition"`
		PostalCode      NullableField[string]  `json:"postal_code"`
		City            NullableField[string]  `json:"city"`
		State           NullableField[string]  `json:"state"`
		Country         NullableField[string]  `json:"country"`
		Lon             NullableField[float64] `json:"lon"`
		Lat             NullableField[float64] `json:"lat"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	argPos = addUpdateClauseNullable("name", payload.Name, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("description", payload.Description, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("legal_form", payload.LegalForm, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("contact_email", payload.ContactEmail, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("contact_phone", payload.ContactPhone, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("web_link", payload.WebLink, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("street", payload.Street, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("house_number", payload.HouseNumber, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("address_addition", payload.AddressAddition, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("postal_code", payload.PostalCode, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("city", payload.City, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("state", payload.State, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("country", payload.Country, &setClauses, &args, argPos)

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
	apiRequest.SetMeta("field_count", len(setClauses))

	query := fmt.Sprintf(`UPDATE %s.organization SET %s WHERE uuid = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)

	args = append(args, orgUuid) // eventId is the last parameter

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		res, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to update organization: %v", err),
			}
		}

		if res.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  errors.New("organization not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "organization", []string{orgUuid})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("refresh projection tables failed"),
			}
		}

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "organization fields updated")
}
