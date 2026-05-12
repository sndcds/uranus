package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

// TODO: Add filter options, e.g. lat/lon/radius

func (h *ApiHandler) GetOrgs(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get organizations")
	ctx := gc.Request.Context()

	type OrganizationResult struct {
		Id           int     `json:"id"`
		Name         *string `json:"name"`
		City         *string `json:"city"`
		Country      *string `json:"country"`
		ContactEmail *string `json:"contact_email"`
	}

	searchStr := strings.TrimSpace(gc.Query("search"))
	if len(searchStr) < 1 {
		apiRequest.Required("URL argument search is required")
		return
	}
	apiRequest.SetMeta("search", searchStr)
	var isEmail = strings.Contains(searchStr, "@")

	var query string
	if isEmail {
		query = fmt.Sprintf("SELECT id, name, city, country, contact_email FROM %s.organization WHERE contact_email ILIKE $1;", h.DbSchema)
	} else {
		query = fmt.Sprintf("SELECT id, name, city, country, contact_email FROM %s.organization WHERE name ILIKE $1;", h.DbSchema)
	}

	searchPattern := "%" + searchStr + "%"
	rows, err := h.DbPool.Query(ctx, query, searchPattern)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	var organizations []OrganizationResult
	for rows.Next() {
		var o OrganizationResult
		if err := rows.Scan(&o.Id, &o.Name, &o.City, &o.Country, &o.ContactEmail); err != nil {
			apiRequest.InternalServerError()
			return
		}
		organizations = append(organizations, o)
	}
	if len(organizations) == 0 {
		apiRequest.NotFound("no organizations found")
		return
	}
	apiRequest.SetMeta("organization_count", len(organizations))
	apiRequest.Success(http.StatusOK, organizations, "")
}
