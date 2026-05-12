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
func (h *ApiHandler) AdminGetChoosableOrgs(gc *gin.Context) {
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	query := app.UranusInstance.SqlAdminChoosableOrgs
	rows, err := h.DbPool.Query(ctx, query, userUuid)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organization struct {
		Uuid string  `json:"uuid"`
		Name *string `json:"name"`
	}

	var organizations []Organization

	for rows.Next() {
		var organization Organization
		err := rows.Scan(&organization.Uuid, &organization.Name)
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
