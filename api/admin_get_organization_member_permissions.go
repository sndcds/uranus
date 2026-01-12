// File: admin_get_organization_member_permissions.go
package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

// Permission to use endpoint checked, 2025-12-23, Roald

func (h *ApiHandler) AdminGetOrganizationMemberPermissions(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	memberId, ok := ParamInt(gc, "memberId") // ID of the user whose permissions are being requested
	if !ok {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "userId is required"})
		return
	}

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "contextId is required"})
		return
	}

	var memberUserId int
	var memberUserDisplayName *string
	var permissions int64

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Check if user can manage member permissions as the organization
		organizationPermissions, err := h.GetUserOrganizationPermissions(gc, tx, userId, organizationId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
			}
		}
		if !organizationPermissions.Has(app.PermManagePermissions) {
			return &ApiTxError{
				Code: http.StatusForbidden,
				Err:  fmt.Errorf("Insufficient permissions"),
			}
		}

		userIdQuery := fmt.Sprintf(`
SELECT oml.user_id, u.display_name
FROM %s.organization_member_link oml
JOIN %s.user u ON u.id = oml.user_id
WHERE oml.organization_id = $1 AND oml.id = $2`,
			h.DbSchema, h.DbSchema)
		err = tx.QueryRow(ctx, userIdQuery, organizationId, memberId).Scan(&memberUserId, &memberUserDisplayName)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
			}
		}

		query := fmt.Sprintf(
			`SELECT permissions FROM %s.user_organization_link WHERE user_id = $1 AND organization_id = $2`,
			h.Config.DbSchema)

		err = tx.QueryRow(ctx, query, memberUserId, organizationId).Scan(&permissions)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("No permissions found for user %d in organization %d", memberUserId, organizationId),
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
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(
		http.StatusOK,
		gin.H{
			"permissions":       permissions,
			"user_id":           memberUserId,
			"user_display_name": memberUserDisplayName,
		})
}
