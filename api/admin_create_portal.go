package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreatePortal(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-create-portal")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	type Payload struct {
		OrgUuid    string `json:"org_uuid" binding:"required"`
		PortalName string `json:"portal_name" binding:"required"`
	}
	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		apiRequest.PayloadError()
		return
	}

	portalName := strings.TrimSpace(payload.PortalName)
	if portalName == "" {
		apiRequest.Error(http.StatusBadRequest, "portal_name cannot be empty")
		return
	}

	apiRequest.Metadata["org_uuid"] = payload.OrgUuid
	apiRequest.Metadata["portal_name"] = payload.PortalName

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckAllOrganizationPermissionsTx(gc, tx, userUuid, payload.OrgUuid, app.UserPermAddPortal)
		if txErr != nil {
			debugf(txErr.Error())
			return txErr
		}

		portalUuid, err := grains_uuid.Uuidv7String()
		apiRequest.Metadata["portal_uuid"] = portalUuid
		query := fmt.Sprintf(`
			INSERT INTO %s.portal (uuid, venue_uuid, name)
			VALUES ($1::uuid, $2)`,
			h.DbSchema)
		_, err = tx.Exec(ctx, query, portalUuid, portalName)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}
		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusCreated, "portal successfully created")
}
