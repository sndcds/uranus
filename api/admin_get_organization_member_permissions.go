// File: admin_get_organization_member_permissions.go
package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// Permission note:
// - Caller must be authenticated
// - Caller must have PermManagePermissions for the given organization
// - Endpoint returns the permission bitmask of an organization member
//
// Permission check enforced via GetUserOrganizationPermissions.
// Verified: 2026-01-11, Roald

func (h *ApiHandler) AdminGetOrganizationMemberPermissions(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-organization-member-permissions")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	memberUuid := gc.Param("memberUuid")
	if memberUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "memberUuid is required")
		return
	}

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "orgUuid is required")
		return
	}

	var memberUserUuid string
	var memberUserDisplayName *string
	var permissions int64

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckOrganizationPermission(gc, tx, userUuid, orgUuid, app.PermManagePermissions)
		if txErr != nil {
			return txErr
		}

		userIdQuery := fmt.Sprintf(`
			SELECT oml.user_uuid, u.display_name
			FROM %s.organization_member_link oml
			JOIN %s.user u ON u.uuid = oml.user_uuid
			WHERE oml.org_uuid = $1::uuid AND oml.user_uuid = $2::uuid`,
			h.DbSchema, h.DbSchema)
		err := tx.QueryRow(ctx, userIdQuery, orgUuid, memberUuid).Scan(&memberUserUuid, &memberUserDisplayName)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("No member with id %d found in organization %d", memberUuid, orgUuid),
				}
			}
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
			}
		}

		query := fmt.Sprintf(
			`SELECT permissions FROM %s.user_organization_link WHERE user_uuid = $1::uuid AND org_uuid = $2::uuid`,
			h.DbSchema)

		err = tx.QueryRow(ctx, query, memberUserUuid, orgUuid).Scan(&permissions)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("No permissions found for user %d in organization %d", memberUserUuid, orgUuid),
				}
			}
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
			}
		}

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK,
		gin.H{
			"user_uuid":         memberUserUuid,
			"user_display_name": memberUserDisplayName,
			"permissions":       permissions,
		},
		"")
}
