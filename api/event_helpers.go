package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func allIDsExist(gc *gin.Context, tx pgx.Tx, table string, ids []int) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	schema := app.Singleton.Config.DbSchema
	var sql string

	switch table {
	case "event_type":
		sql = fmt.Sprintf("SELECT COUNT(DISTINCT type_id) = $2 FROM %s.event_type WHERE type_id = ANY($1)", schema)
	case "event_genre":
		sql = fmt.Sprintf("SELECT COUNT(DISTINCT type_id) = $2 FROM %s.genre_type WHERE type_id = ANY($1)", schema)
	default:
		return false, fmt.Errorf("unsupported table for ID check: %s", table)
	}

	fmt.Println(sql)
	fmt.Println(ids)
	fmt.Println(len(ids))

	var allExist bool
	err := tx.QueryRow(gc, sql, ids, len(ids)).Scan(&allExist)
	if err != nil {
		return false, err
	}
	return allExist, nil
}

func UserCanEditEvent(gc *gin.Context, tx pgx.Tx, eventId int) (bool, error) {
	userId, err := app.CurrentUserId(gc)
	if userId < 0 {
		return false, err
	}

	schema := app.Singleton.Config.DbSchema

	query := fmt.Sprintf(`
		SELECT EXISTS (
			-- Case 1: via organizer
			SELECT 1
			FROM %[1]s.event e
			JOIN %[1]s.user_organizer_links uol ON e.organizer_id = uol.organizer_id
			JOIN %[1]s.user_role ur ON uol.user_role_id = ur.id
			WHERE e.id = $1 AND uol.user_id = $2 AND ur.edit_event = TRUE

			UNION

			-- Case 2: via direct event link
			SELECT 1
			FROM %[1]s.user_event_links uel
			JOIN %[1]s.user_role ur2 ON uel.user_role_id = ur2.id
			WHERE uel.event_id = $1 AND uel.user_id = $2 AND ur2.edit_event = TRUE
		) AS can_edit;
	`, schema)

	var canEdit bool
	err = tx.QueryRow(gc, query, eventId, userId).Scan(&canEdit)
	if err != nil {
		return false, fmt.Errorf("failed to check event edit permission: %w", err)
	}

	return canEdit, nil
}
