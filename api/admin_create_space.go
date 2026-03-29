package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreateSpace(gc *gin.Context) {
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-create-space")

	type Payload struct {
		OrgUuid   string `json:"org_uuid" binding:"required"`
		VenueUuid string `json:"venue_uuid" binding:"required"`
		SpaceName string `json:"space_name" binding:"required"`
	}
	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		return
	}

	spaceName := strings.TrimSpace(payload.SpaceName)
	if spaceName == "" {
		apiRequest.Error(http.StatusBadRequest, "space_name cannot be empty")
		return
	}

	apiRequest.Metadata["prganization_id"] = payload.OrgUuid
	apiRequest.Metadata["venue_id"] = payload.VenueUuid
	apiRequest.Metadata["space_name"] = spaceName

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckOrganizationAllPermissions(
			gc, tx, userUuid, payload.OrgUuid,
			app.PermAddSpace)
		if txErr != nil {
			return txErr
		}

		newSpaceId := -1
		query := fmt.Sprintf(`INSERT INTO %s.space (venue_id, name) VALUES ($1, $2) RETURNING id`, h.DbSchema)
		err := tx.QueryRow(ctx, query, payload.VenueUuid, spaceName).Scan(&newSpaceId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("Internal server error"),
			}
		}
		apiRequest.Metadata["space_id"] = newSpaceId
		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "space successfully created")
}
