package api

// TODO: Review code

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetUserEffectiveVenuePermissions(
	gc *gin.Context,
	tx pgx.Tx,
	userId int,
	venueId int,
) (app.Permission, error) {

	ctx := gc.Request.Context()
	var result pgtype.Int8

	err := tx.QueryRow(
		ctx,
		app.UranusInstance.SqlGetUserEffectiveVenuePermissions,
		userId,
		venueId,
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

// GetUserOrganizationPermissions returns the permissions a user has for an organization.
func (h *ApiHandler) GetUserOrganizationPermissions(
	gc *gin.Context,
	tx pgx.Tx,
	userId int,
	organizationId int,
) (app.Permission, error) {

	ctx := gc.Request.Context()
	var result pgtype.Int8

	err := tx.QueryRow(
		ctx, app.UranusInstance.SqlGetUserOrganizationPermissions,
		userId, organizationId,
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

func (h *ApiHandler) IsSpaceInVenue(
	gc *gin.Context,
	tx pgx.Tx,
	spaceId int,
	venueId int,
) (bool, error) {

	ctx := gc.Request.Context()

	var result bool
	query := fmt.Sprintf(
		`SELECT EXISTS (SELECT 1 FROM %s.space WHERE id = $1 AND venue_id = $2) AS space_exist`,
		h.DbSchema)
	err := tx.QueryRow(ctx, query, spaceId, venueId).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return result, nil
}

func (h *ApiHandler) GetOrganizationIdByEvenId(
	gc *gin.Context,
	tx pgx.Tx,
	eventId int,
) (int, error) {

	ctx := gc.Request.Context()

	query := fmt.Sprintf(`SELECT e.organization_id FROM %s.event e WHERE e.id = $1`, h.Config.DbSchema)
	orgId := -1
	err := tx.QueryRow(ctx, query, eventId).Scan(&orgId)
	if err != nil {
		return -1, err
	}

	return orgId, nil
}

func (h *ApiHandler) GetOrganizationIdByEventDateId(
	gc *gin.Context,
	tx pgx.Tx,
	eventDateId int,
) (int, error) {

	ctx := gc.Request.Context()

	query := fmt.Sprintf(`
			SELECT e.organization_id FROM %s.event_date ed JOIN %s.event e ON e.id = ed.event_id WHERE ed.id = $1`,
		h.Config.DbSchema, h.Config.DbSchema)
	orgId := -1
	err := tx.QueryRow(ctx, query, eventDateId).Scan(&orgId)
	if err != nil {
		return -1, err
	}

	return orgId, nil
}

// CheckOrganizationPermission verifies if a user has a specific permission
// in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckOrganizationPermission(
	gc *gin.Context,
	tx pgx.Tx,
	userId, organizationId int,
	perm app.Permission,
) *ApiTxError {
	organizationPermissions, err := h.GetUserOrganizationPermissions(gc, tx, userId, organizationId)
	if err != nil {
		return &ApiTxError{
			Code: http.StatusInternalServerError,
			Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
		}
	}

	if !organizationPermissions.Has(perm) {
		return &ApiTxError{
			Code: http.StatusForbidden,
			Err:  fmt.Errorf("Insufficient permissions"),
		}
	}

	return nil
}

// CheckOrganizationAllPermissions verifies if a user has all of the specified
// permissions in the given organization. Returns an ApiTxError if the check fails.
func (h *ApiHandler) CheckOrganizationAllPermissions(
	gc *gin.Context,
	tx pgx.Tx,
	userId, organizationId int,
	permMask app.Permission,
) *ApiTxError {
	organizationPermissions, err := h.GetUserOrganizationPermissions(gc, tx, userId, organizationId)
	if err != nil {
		return &ApiTxError{
			Code: http.StatusInternalServerError,
			Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
		}
	}

	if !organizationPermissions.HasAll(permMask) {
		return &ApiTxError{
			Code: http.StatusForbidden,
			Err:  fmt.Errorf("Insufficient permissions"),
		}
	}

	return nil
}
