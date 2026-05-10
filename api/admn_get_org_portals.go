package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// PermissionChecks: TODO
// Verified: TODO

func (h *ApiHandler) AdminGetOrgPortals(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-user-org-portals-list")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("missing parameter orgUuid")
		return
	}

	var portals []model.AdminListPortal
	var orgPermissions app.Permissions

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		var err error

		rows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrgPortals, orgUuid, userUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error: %v", err),
			}
		}
		defer rows.Close()

		for rows.Next() {
			var permissions app.Permissions
			var p model.AdminListPortal
			err := rows.Scan(
				&p.Uuid,
				&p.Name,
				&p.Description,
				&permissions,
			)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("Internal server error: %v", err),
				}
			}

			if permissions.Has(app.UserPermEditPortal) {
				p.CanEditPortal = true
			}
			if permissions.Has(app.UserPermDeletePortal) {
				p.CanDeletePortal = true
			}
		}

		orgPermissions, err = h.GetUserOrganizationPermissionsTx(gc, tx, userUuid, orgUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	canAddPortal := orgPermissions.Has(app.UserPermAddPortal)
	apiRequest.SetMeta("can_add_portal", canAddPortal)
	apiRequest.SetMeta("total_portals", len(portals))
	apiRequest.Success(http.StatusOK, portals, "")
}
