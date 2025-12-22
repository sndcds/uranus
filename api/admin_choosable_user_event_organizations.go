package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// AdminChoosableUserEventOrganizations returns a list of event organizations
// that can be selected (choosable) by an admin user. It responds with a JSON
// array of items.
//
// This endpoint is intended for administrative use only and may require
// authentication or specific permissions.
func (h *ApiHandler) AdminChoosableUserEventOrganizations(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	// Parse organization Id from path param
	organizationIdStr := gc.Param("organizationId")
	organizationId, err := strconv.Atoi(organizationIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := app.Singleton.SqlAdminChoosableUserEventOrganizations
	rows, err := h.DbPool.Query(ctx, query, userId, organizationId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organization struct {
		Id          int64   `json:"id"`
		Name        *string `json:"name"`
		City        *string `json:"city"`
		CountryCode *string `json:"country_code"`
	}

	var organizations []Organization

	for rows.Next() {
		var organization Organization
		err := rows.Scan(&organization.Id, &organization.Name, &organization.City, &organization.CountryCode)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizations = append(organizations, organization)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(organizations) == 0 {
		gc.JSON(http.StatusOK, []Organization{}) // Returns empty array
		return
	}

	gc.JSON(http.StatusOK, organizations)
}
