package api

// TODO: Review code

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetUserEffectiveVenuePermissionsTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	venueUuid string,
) (app.Permission, error) {
	ctx := gc.Request.Context()
	var result pgtype.Int8

	err := tx.QueryRow(
		ctx,
		app.UranusInstance.SqlGetUserEffectiveVenuePermissions,
		userUuid,
		venueUuid,
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

func (h *ApiHandler) IsSpaceInVenueTx(
	gc *gin.Context,
	tx pgx.Tx,
	spaceUuid string,
	venueUuid string,
) (bool, error) {
	ctx := gc.Request.Context()

	var result bool
	query := fmt.Sprintf(
		`SELECT EXISTS (SELECT 1 FROM %s.space WHERE uuid = $1 AND venue_uuid = $2) AS space_exist`,
		h.DbSchema)
	err := tx.QueryRow(ctx, query, spaceUuid, venueUuid).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return result, nil
}
