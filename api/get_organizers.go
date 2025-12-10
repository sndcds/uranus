package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

type Organizer struct {
	ID           int     `json:"id"`
	Name         *string `json:"name"`
	City         *string `json:"city"`
	CountryCode  *string `json:"country_code"`
	ContactEmail *string `json:"contact_email"`
}

func (h *ApiHandler) GetOrganizers(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	searchStr := strings.TrimSpace(gc.Query("search"))
	if len(searchStr) < 1 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "URL argument search is required"})
		return
	}
	var isEmail = strings.Contains(searchStr, "@")

	var query string
	if isEmail {
		query = fmt.Sprintf("SELECT id, name, city, country_code, contact_email FROM %s.organizer WHERE contact_email ILIKE $1;", h.Config.DbSchema)
	} else {
		query = fmt.Sprintf("SELECT id, name, city, country_code, contact_email FROM %s.organizer WHERE name ILIKE $1;", h.Config.DbSchema)
	}

	searchPattern := "%" + searchStr + "%"
	rows, err := pool.Query(ctx, query, searchPattern)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var organizers []Organizer
	for rows.Next() {
		var o Organizer
		if err := rows.Scan(&o.ID, &o.Name, &o.City, &o.CountryCode, &o.ContactEmail); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizers = append(organizers, o)
	}

	gc.JSON(http.StatusOK, organizers)
}
