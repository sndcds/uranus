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

func (h *ApiHandler) AdminCreateSpace(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-create-space")

	type Payload struct {
		OrganizationId int    `json:"organization_id" binding:"required"`
		VenueId        int    `json:"venue_id" binding:"required"`
		SpaceName      string `json:"space_name" binding:"required"`
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

	apiRequest.Metadata["prganization_id"] = payload.OrganizationId
	apiRequest.Metadata["venue_id"] = payload.VenueId
	apiRequest.Metadata["space_name"] = spaceName

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckOrganizationAllPermissions(
			gc, tx, userId, payload.OrganizationId,
			app.PermAddSpace)
		if txErr != nil {
			return txErr
		}

		newSpaceId := -1
		query := fmt.Sprintf(`INSERT INTO %s.space (venue_id, name) VALUES ($1, $2) RETURNING id`, h.DbSchema)
		err := tx.QueryRow(ctx, query, payload.VenueId, spaceName).Scan(&newSpaceId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error"),
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
