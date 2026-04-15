package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreateVenue(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-create-venue")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	type IncomingPayload struct {
		OrgUuid   string `json:"org_uuid" binding:"required"`
		VenueName string `json:"venue_name" binding:"required"`
	}
	payload, ok := grains_api.DecodeJSONBody[IncomingPayload](gc, apiRequest)
	if !ok {
		apiRequest.InvalidJSONInput()
		return
	}

	venueName := strings.TrimSpace(payload.VenueName)
	if venueName == "" {
		apiRequest.Error(http.StatusBadRequest, "venue_name cannot be empty")
		return
	}

	apiRequest.Metadata["org_uuid"] = payload.OrgUuid
	apiRequest.Metadata["venue_name"] = venueName

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckOrganizationAllPermissions(
			gc, tx, userUuid, payload.OrgUuid,
			app.PermAddVenue)
		if txErr != nil {
			return txErr
		}

		venueUuid, err := grains_uuid.Uuidv7String()
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to generate uuid: %v", err),
			}
		}

		query := fmt.Sprintf(`INSERT INTO %s.venue (uuid, org_uuid, name) VALUES ($1::uuid, $2::uuid, $3)`, h.DbSchema)
		_, err = tx.Exec(ctx, query, venueUuid, payload.OrgUuid, venueName)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("Internal server error"),
			}
		}
		apiRequest.Metadata["venue_uuid"] = venueUuid
		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "venue successfully created")
}
