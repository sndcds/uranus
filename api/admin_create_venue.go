package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreateVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-create-venue")

	type Payload struct {
		OrganizationId int    `json:"organization_id" binding:"required"`
		VenueName      string `json:"venue_name" binding:"required"`
	}
	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		return
	}

	venueName := strings.TrimSpace(payload.VenueName)
	if venueName == "" {
		apiRequest.Error(http.StatusBadRequest, "venue_name cannot be empty")
		return
	}

	apiRequest.Metadata["organization_id"] = payload.OrganizationId
	apiRequest.Metadata["venue_name"] = venueName

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckOrganizationAllPermissions(
			gc, tx, userId, payload.OrganizationId,
			app.PermAddVenue)
		if txErr != nil {
			return txErr
		}

		newVenueId := -1
		query := fmt.Sprintf(`INSERT INTO %s.venue (organization_id, name) VALUES ($1, $2) RETURNING id`, h.DbSchema)
		err := tx.QueryRow(ctx, query, payload.OrganizationId, venueName).Scan(&newVenueId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error"),
			}
		}
		apiRequest.Metadata["venue_id"] = newVenueId
		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "venue successfully created")
}
