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

func (h *ApiHandler) GetOrgUuidByEventUuidTx(
	gc *gin.Context,
	tx pgx.Tx,
	eventUuid string,
) (string, error) {
	ctx := gc.Request.Context()
	query := fmt.Sprintf(`SELECT e.org_uuid FROM %s.event e WHERE e.uuid = $1::uuid`, h.DbSchema)
	orgUuid := ""
	err := tx.QueryRow(ctx, query, eventUuid).Scan(&orgUuid)
	if err != nil {
		return "", err
	}

	return orgUuid, nil
}

func (h *ApiHandler) GetOrgUuidByEventDateUuidTx(
	gc *gin.Context,
	tx pgx.Tx,
	eventDateUuid string,
) (string, error) {
	ctx := gc.Request.Context()
	query := fmt.Sprintf(`
			SELECT e.org_uuid FROM %s.event_date ed JOIN %s.event e ON e.uuid = ed.event_uuid WHERE ed.uuid = $1::uuid`,
		h.DbSchema, h.DbSchema)
	orgUuid := ""
	err := tx.QueryRow(ctx, query, eventDateUuid).Scan(&orgUuid)
	if err != nil {
		return "", err
	}

	return orgUuid, nil
}

func (h *ApiHandler) GetOrgUuidBySpaceUuidTx(
	gc *gin.Context,
	tx pgx.Tx,
	spaceUuid string,
) (string, error) {
	ctx := gc.Request.Context()
	query := fmt.Sprintf(`
		SELECT v.org_uuid
		FROM %s.space s
		JOIN %s.venue v ON v.uuid = s.venue_uuid
		WHERE s.uuid = $1::uuid
		`,
		h.DbSchema, h.DbSchema)
	orgUuid := ""
	err := tx.QueryRow(ctx, query, spaceUuid).Scan(&orgUuid)
	if err != nil {
		return "", err
	}

	return orgUuid, nil
}

// CheckOrgPermissionTx verifies if a user has a specific permission
// in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckOrgPermissionTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	orgUuid string,
	perm app.Permissions,
) *ApiTxError {
	orgPermissions, err := h.GetUserOrgPermissionsTx(gc, tx, userUuid, orgUuid)
	if err != nil {
		return &ApiTxError{
			Code: http.StatusInternalServerError,
			Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
		}
	}

	if !orgPermissions.Has(perm) {
		return &ApiTxError{
			Code: http.StatusForbidden,
			Err:  errors.New("Insufficient permissions"),
		}
	}

	return nil
}

// CheckAllOrgPermissionsTx verifies if a user has all of the specified
// permissions in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckAllOrgPermissionsTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	orgUuid string,
	permMask app.Permissions,
) *ApiTxError {
	orgPermissions, err := h.GetUserOrgPermissionsTx(gc, tx, userUuid, orgUuid)
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

// GetUserOrgPermissionsTx returns the permissions a user has for an organization.
func (h *ApiHandler) GetUserOrgPermissionsTx(
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
