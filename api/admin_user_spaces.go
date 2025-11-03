package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func AdminUserSpacesHandler(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	modeStr := gc.Param("mode")
	fmt.Println("modeStr:", modeStr)

	type EditableSpace struct {
		OrganizerID   int    `json:"organizer_id"`
		OrganizerName string `json:"organizer_name"`
		VenueID       int    `json:"venue_id"`
		VenueName     string `json:"venue_name"`
		SpaceID       int    `json:"space_id"`
		SpaceName     string `json:"space_name"`
	}

	var rows pgx.Rows
	var err error
	switch modeStr {
	case "can-add-event":
		sql := app.Singleton.SqlAdminSpacesCanAddEvent
		rows, err = db.Query(ctx, sql, userId)
		break
	case "for-event":
		spaceID, hasSpaceID := GetContextParameterAsInt(gc, "space-id")
		fmt.Println("hasSpaceID:", hasSpaceID, "spaceID:", spaceID)
		if !hasSpaceID {
			gc.JSON(http.StatusBadRequest, gin.H{"message": "event-id missing"})
			return
		}
		sql := app.Singleton.SqlAdminSpacesForEvent
		rows, err = db.Query(ctx, sql, userId, spaceID)
		break
	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
		return
	}

	if err != nil {
		fmt.Println("err:", err.Error())
		gc.JSON(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	spaces := []EditableSpace{}

	for rows.Next() {
		var sp EditableSpace
		err := rows.Scan(
			&sp.OrganizerID,
			&sp.OrganizerName,
			&sp.VenueID,
			&sp.VenueName,
			&sp.SpaceID,
			&sp.SpaceName,
		)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		spaces = append(spaces, sp)
	}

	gc.JSON(http.StatusOK, spaces)
}
