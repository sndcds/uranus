package api

// TODO: Review code

import (
	"fmt"

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
		app.Singleton.SqlGetUserEffectiveVenuePermissions,
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
		ctx, app.Singleton.SqlGetUserOrganizationPermissions,
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

// GetUserVenuePermissions returns the permissions a user has for a venue.
func (h *ApiHandler) GetUserVenuePermissions(
	gc *gin.Context,
	tx pgx.Tx,
	userId int,
	venueId int,
) (app.Permission, error) {
	ctx := gc.Request.Context()
	var result pgtype.Int8

	err := tx.QueryRow(
		ctx, app.Singleton.SqlGetUserVenuePermissions,
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

func (h *ApiHandler) IsSpaceInVenue(
	gc *gin.Context,
	tx pgx.Tx,
	spaceId int,
	venueId int,
) (bool, error) {
	ctx := gc.Request.Context()
	var result bool
	query := fmt.Sprintf(
		`SELECT EXISTS (SELECT 1 FROM uranus.space WHERE id = $1 AND venue_id = $2) AS space_exist`,
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
