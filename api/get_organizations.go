package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

type Organization struct {
	ID           int     `json:"id"`
	Name         *string `json:"name"`
	City         *string `json:"city"`
	CountryCode  *string `json:"country_code"`
	ContactEmail *string `json:"contact_email"`
}

func (h *ApiHandler) GetOrganizations(gc *gin.Context) {
	ctx := gc.Request.Context()

	searchStr := strings.TrimSpace(gc.Query("search"))
	if len(searchStr) < 1 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "URL argument search is required"})
		return
	}
	var isEmail = strings.Contains(searchStr, "@")

	var query string
	if isEmail {
		query = fmt.Sprintf("SELECT id, name, city, country_code, contact_email FROM %s.organization WHERE contact_email ILIKE $1;", h.Config.DbSchema)
	} else {
		query = fmt.Sprintf("SELECT id, name, city, country_code, contact_email FROM %s.organization WHERE name ILIKE $1;", h.Config.DbSchema)
	}

	searchPattern := "%" + searchStr + "%"
	rows, err := h.DbPool.Query(ctx, query, searchPattern)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var organizations []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.City, &o.CountryCode, &o.ContactEmail); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizations = append(organizations, o)
	}

	gc.JSON(http.StatusOK, organizations)
}
