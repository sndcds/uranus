package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// The endpoint returns space details only if the authenticated user
// is linked to the space (via the SQL query).
// PermissionChecks: Enforced in SQL; no additional checks needed in Go.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetSpace(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-space")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	spaceUuid := gc.Param("spaceUuid")
	if spaceUuid == "" {
		apiRequest.Required("spaceUuid is required")
		return
	}
	apiRequest.SetMeta("space_uuid", spaceUuid)

	var space model.Space

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Check user permission
		orgUuid, err := h.GetOrgUuidBySpaceUuidTx(gc, tx, spaceUuid)
		if err != nil {
			return TxInternalError(err)
		}
		txErr := h.CheckOrgPermissionTx(gc, tx, userUuid, orgUuid, app.UserPermEditSpace)
		if txErr != nil {
			return txErr
		}

		query := app.UranusInstance.SqlAdminGetSpace
		row := tx.QueryRow(ctx, query, spaceUuid, userUuid)
		err = row.Scan(
			&space.Uuid,
			&space.Name,
			&space.Description,
			&space.SpaceType,
			&space.BuildingLevel,
			&space.TotalCapacity,
			&space.SeatingCapacity,
			&space.WebLink,
			&space.AccessibilityFlags,
			&space.AccessibilitySummary,
			&space.AreaSqm,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code:    http.StatusNotFound,
					Message: "space not found",
					Err:     err,
				}
			}
			return &ApiTxError{
				Code:    http.StatusInternalServerError,
				Message: "failed to load space",
				Err:     err,
			}
		}
		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Message)
		return
	}

	apiRequest.Success(http.StatusOK, space, "Space loaded successfully")
}
