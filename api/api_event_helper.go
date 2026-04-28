package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetUserEventPermissionsTx(
	gc *gin.Context,
	tx pgx.Tx,
	userUuid string,
	eventUuid string,
) (app.Permission, error) {

	debugf("a")
	ctx := gc.Request.Context()
	var permissions pgtype.Int8

	debugf("b")
	err := tx.QueryRow(
		ctx,
		app.UranusInstance.SqlGetUserEventPermissions,
		userUuid,
		eventUuid,
	).Scan(&permissions)
	if err != nil {
		debugf(err.Error())
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	debugf("c")
	if !permissions.Valid {
		return 0, nil
	}
	debugf("d")

	return app.Permission(permissions.Int64), nil
}

func (h *ApiHandler) GetUserEventPermissions(
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
