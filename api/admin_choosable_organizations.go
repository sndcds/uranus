package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: Only returns choosable organizations for the authenticated user.
// PermissionChecks: Unnecessary.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetChoosableOrganizations(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	query := app.UranusInstance.SqlAdminChoosableOrganizations
	rows, err := h.DbPool.Query(ctx, query, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organization struct {
		Id   int64   `json:"id"`
		Name *string `json:"name"`
	}

	var organizations []Organization

	for rows.Next() {
		var organization Organization
		err := rows.Scan(&organization.Id, &organization.Name)
		if err != nil {
			fmt.Println(err.Error())
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

	result := map[string]interface{}{
		"organizations": organizations,
		"total_count":   len(organizations),
	}

	gc.JSON(http.StatusOK, result)
}
