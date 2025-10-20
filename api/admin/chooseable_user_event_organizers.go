package api_admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
)

func ChoosableUserEventOrganizersHandler(gc *gin.Context) {
	organizerID, _ := api.GetContextParameterAsInt(gc, "organizer-id")

	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	userID, err := app.CurrentUserId(gc)
	if userID < 0 {
		gc.JSON(http.StatusUnauthorized, err)
		return
	}

	var rows pgx.Rows
	sql := app.Singleton.SqlAdminChoosableUserEventOrganizers
	rows, err = db.Query(ctx, sql, userID, organizerID)

	if err != nil {
		gc.JSON(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type UserEventOrganizer struct {
		OrganizerID      int64  `json:"organizer_id"`
		OrganizerName    string `json:"organizer_name"`
		OrganizerCity    string `json:"organizer_city"`
		OrganizerCountry string `json:"organizer_country"`
		OrganizerWebURL  string `json:"organizer_web_url"`
	}

	var organizers []UserEventOrganizer

	for rows.Next() {
		var ueo UserEventOrganizer
		if err := rows.Scan(
			&ueo.OrganizerID,
			&ueo.OrganizerName,
			&ueo.OrganizerCity,
			&ueo.OrganizerCountry,
			&ueo.OrganizerWebURL,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizers = append(organizers, ueo)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Wrap in outer object with metadata
	response := gin.H{
		"api":              "Uranus",
		"version":          "1.0.0",
		"event-organizers": organizers,
	}

	gc.JSON(http.StatusOK, response)
}
