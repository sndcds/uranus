package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// Permission note:
// - Caller must be authenticated
// - Caller must have PermManagePermissions for the organization
// - Allows updating permission bits of organization members
// - Caller cannot modify their own ManagePermissions or ManageTeam bits
//
// Permission checks enforced via GetUserOrganizationPermissions.
// Safeguards against self-escalation are applied.
// Verified: 2026-01-11, Roald

// AdminUpdateOrganizationMemberPermission updates a single permission bit
// for a member of an organization.
//
// This endpoint allows an authorized organization administrator to enable
// or disable a specific permission bit for a given organization member.
// Permissions are stored as a 64-bit bitmask and updated using a bitwise
// operation inside a database transaction.
//
// Authentication & Authorization:
//   - Requires an authenticated user (user-id must be set in Gin context).
//   - The authenticated user must have BOTH PermManagePermissions and
//     PermManageTeam permissions for the target organization.
//
// URL Parameters:
//   - organizationId (int): ID of the organization.
//   - memberId (int): ID of the organization member whose permissions
//     will be updated.
//
// Request Body (JSON):
//
//	{
//	  "bit": <int>,       // Permission bit index (0–63)
//	  "enabled": <bool>   // true to enable the bit, false to disable it
//	}
//
// Behavior:
//   - Validates input parameters and JSON payload.
//   - Starts a database transaction.
//   - Verifies the caller’s organization permissions.
//   - Sets or clears the specified permission bit using a bitwise operation.
//   - Commits the transaction on success.
//
// Responses:
//   - 200 OK: Returns the updated permissions bitmask.
//     {
//     "permissions": <int64>
//     }
//   - 400 Bad Request: Missing or invalid parameters or payload.
//   - 403 Forbidden: Caller lacks sufficient permissions.
//   - 404 Not Found: Target member does not exist in the organization.
//   - 500 Internal Server Error: Database or transaction failure.
func (h *ApiHandler) AdminUpdateOrganizationMemberPermission(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "organizationId is required"})
		return
	}

	memberId, ok := ParamInt(gc, "memberId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "memberId is required"})
		return
	}

	var inputReq struct {
		Bit     int  `json:"bit"`
		Enabled bool `json:"enabled"`
	}
	if err := gc.ShouldBindJSON(&inputReq); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if inputReq.Bit < 0 || inputReq.Bit > 63 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "bit must be between 0 and 63"})
		return
	}

	var updatedPermissions int64

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckOrganizationPermission(gc, tx, userId, organizationId, app.PermManagePermissions)
		if txErr != nil {
			return txErr
		}

		// Ckeck if member is the admin user
		var orgMemberLink model.OrganizationMemberLink
		orgMemberLink.Id = memberId
		err := tx.QueryRow(
			ctx, app.UranusInstance.SqlAdminGetOrganizationMemberLink,
			memberId).
			Scan(
				&orgMemberLink.OrganizationId,
				&orgMemberLink.UserId,
				&orgMemberLink.HasJoined,
				&orgMemberLink.InvitedByUserId,
				&orgMemberLink.InvitedAt,
				&orgMemberLink.CreatedAt,
				&orgMemberLink.ModifiedAt)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  fmt.Errorf("Failed to get organization member link, %v", err),
			}
		}

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNoContent,
					Err:  fmt.Errorf("Failed to check membership, %s", err.Error()),
				}
			} else {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("Failed to check membership, %s", err.Error()),
				}
			}
		}

		memberUserId := orgMemberLink.UserId

		// If the user is trying to set their own ManagePermissions or ManageTeam bit, block it
		if memberUserId == userId && (inputReq.Bit == app.PermBitManagePermissions || inputReq.Bit == app.PermBitManageTeam) {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  fmt.Errorf("Bits %d protected", inputReq.Bit),
			}
		}

		// Perform the bitwise update
		bitUpdateQuery := fmt.Sprintf(`
UPDATE %s.user_organization_link
SET permissions = CASE
	WHEN $1 THEN permissions | (1::bigint << $2)
	ELSE permissions & ~(1::bigint << $2)
END
WHERE user_id = $3 AND organization_id = $4
RETURNING permissions`,
			h.Config.DbSchema)

		err = tx.QueryRow(
			ctx, bitUpdateQuery, inputReq.Enabled, inputReq.Bit, memberUserId, organizationId).
			Scan(&updatedPermissions)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("Target member does not exist in the organization"),
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

	gc.JSON(http.StatusOK, gin.H{"permissions": updatedPermissions})
}
