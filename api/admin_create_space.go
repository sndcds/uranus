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

func (h *ApiHandler) AdminCreateSpace(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-create-space")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

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

	apiRequest.Metadata["org_uuid"] = payload.OrgUuid
	apiRequest.Metadata["venue_uuid"] = payload.VenueUuid
	apiRequest.Metadata["space_name"] = spaceName

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckAllOrganizationPermissionsTx(
			gc, tx, userUuid, payload.OrgUuid,
			app.PermAddSpace)
		if txErr != nil {
			return txErr
		}

		spaceUuid, err := grains_uuid.Uuidv7String()
		apiRequest.Metadata["space_uuid"] = spaceUuid
		query := fmt.Sprintf(`INSERT INTO %s.space (uuid, venue_uuid, name) VALUES ($1::uuid, $2::uuid, $3)`, h.DbSchema)
		_, err = tx.Exec(ctx, query, spaceUuid, payload.VenueUuid, spaceName)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("Internal server error"),
			}
		}
		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "space successfully created")
}
