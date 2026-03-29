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

func (h *ApiHandler) GetOrganizationUuidByEvenUuid(
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

func (h *ApiHandler) GetOrganizationIdByEventDateId(
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

// CheckOrganizationPermission verifies if a user has a specific permission
// in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckOrganizationPermission(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	orgUuid string,
	perm app.Permission,
) *ApiTxError {
	organizationPermissions, err := h.GetUserOrganizationPermissions(gc, tx, userUuid, orgUuid)
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

// CheckOrganizationAllPermissions verifies if a user has all of the specified
// permissions in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckOrganizationAllPermissions(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	orgUuid string,
	permMask app.Permission,
) *ApiTxError {
	orgPermissions, err := h.GetUserOrganizationPermissions(gc, tx, userUuid, orgUuid)
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

// GetUserOrganizationPermissions returns the permissions a user has for an organization.
func (h *ApiHandler) GetUserOrganizationPermissions(
	gc *gin.Context,
	tx pgx.Tx,
	userId string,
	orgUuid string,
) (app.Permission, error) {

	ctx := gc.Request.Context()
	var result pgtype.Int8

	err := tx.QueryRow(
		ctx,
		app.UranusInstance.SqlGetUserOrganizationPermissions,
		userId,
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
	return app.Permission(result.Int64), nil
}
