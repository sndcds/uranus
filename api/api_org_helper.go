package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetOrganizationUuidByEvenUuidTx(
	gc *gin.Context,
	tx pgx.Tx,
	eventUuid string,
) (string, error) {

	ctx := gc.Request.Context()

	query := fmt.Sprintf(`SELECT e.org_id FROM %s.event e WHERE e.id = $1`, h.DbSchema)
	orgUuid := ""
	err := tx.QueryRow(ctx, query, eventUuid).Scan(&orgUuid)
	if err != nil {
		return "", err
	}

	return orgUuid, nil
}

func (h *ApiHandler) GetOrganizationIdByEventDateIdTx(
	gc *gin.Context,
	tx pgx.Tx,
	eventDateId int,
) (int, error) {

	ctx := gc.Request.Context()

	query := fmt.Sprintf(`
			SELECT e.org_id FROM %s.event_date ed JOIN %s.event e ON e.id = ed.event_id WHERE ed.id = $1`,
		h.DbSchema, h.DbSchema)
	organizationId := -1
	err := tx.QueryRow(ctx, query, eventDateId).Scan(&organizationId)
	if err != nil {
		return -1, err
	}

	return organizationId, nil
}

// CheckOrganizationPermissionTx verifies if a user has a specific permission
// in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckOrganizationPermissionTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	orgUuid string,
	perm app.Permissions,
) *ApiTxError {
	organizationPermissions, err := h.GetUserOrganizationPermissionsTx(gc, tx, userUuid, orgUuid)
	if err != nil {
		return &ApiTxError{
			Code: http.StatusInternalServerError,
			Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
		}
	}

	if !organizationPermissions.Has(perm) {
		return &ApiTxError{
			Code: http.StatusForbidden,
			Err:  errors.New("Insufficient permissions"),
		}
	}

	return nil
}

// CheckAllOrganizationPermissionsTx verifies if a user has all of the specified
// permissions in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckAllOrganizationPermissionsTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	orgUuid string,
	permMask app.Permissions,
) *ApiTxError {
	orgPermissions, err := h.GetUserOrganizationPermissionsTx(gc, tx, userUuid, orgUuid)
	if err != nil {
		return &ApiTxError{
			Code: http.StatusInternalServerError,
			Err:  errors.New("Transaction failed"),
		}
	}

	if !orgPermissions.HasAll(permMask) {
		return &ApiTxError{
			Code: http.StatusForbidden,
			Err:  errors.New("Insufficient permissions"),
		}
	}

	return nil
}

// GetUserOrganizationPermissionsTx returns the permissions a user has for an organization.
func (h *ApiHandler) GetUserOrganizationPermissionsTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	orgUuid string,
) (app.Permissions, error) {

	ctx := gc.Request.Context()
	var result pgtype.Int8

	err := tx.QueryRow(
		ctx,
		app.UranusInstance.SqlGetUserOrgPermissions,
		userUuid,
		orgUuid,
	).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	if !result.Valid {
		return 0, nil
	}
	return app.Permissions(result.Int64), nil
}
