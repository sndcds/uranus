package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sndcds/uranus/app"
)

// GetUserEventOrganizerPermissionsTx fetches a user's permission for the organizer of an event within a transaction.
func (h *ApiHandler) GetUserEventOrganizerPermissionsTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	eventUuid string,
) (app.Permission, error) {

	ctx := gc.Request.Context()
	var permissions pgtype.Int8

	err := tx.QueryRow(
		ctx,
		app.UranusInstance.SqlGetUserEventPermissions,
		userUuid,
		eventUuid,
	).Scan(&permissions)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	if !permissions.Valid {
		return 0, nil
	}

	return app.Permission(permissions.Int64), nil
}

// GetUserEventOrganizerPermissions fetches a user's permission for the organizer of an event.
func (h *ApiHandler) GetUserEventOrganizerPermissions(
	gc *gin.Context,
	userUuid string,
	eventUuid string,
) (app.Permission, error) {

	ctx := gc.Request.Context()
	var permissions pgtype.Int8

	err := h.DbPool.QueryRow(
		ctx,
		app.UranusInstance.SqlGetUserEventPermissions,
		userUuid,
		eventUuid,
	).Scan(&permissions)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	if !permissions.Valid {
		return 0, nil
	}

	return app.Permission(permissions.Int64), nil
}
