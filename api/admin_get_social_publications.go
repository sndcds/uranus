package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetSocialPublications(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-social-publications")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)
	if !app.ValidUUID(userUuid) {
		apiRequest.Error(http.StatusUnauthorized, "authentication required")
		return
	}

	query := h.socialPublicationQuery() + fmt.Sprintf(`
		JOIN %s.social_post s ON s.uuid = p.social_post_uuid
		WHERE EXISTS (SELECT 1 FROM %s.user_organization_link l
			JOIN %s.organization_member_link m
				ON m.org_uuid = l.org_uuid AND m.user_uuid = l.user_uuid AND m.has_joined = true
			WHERE l.org_uuid = s.org_uuid AND l.user_uuid = $1
			AND (l.permissions & $2) = $2)`, h.DbSchema, h.DbSchema, h.DbSchema)
	args := []any{userUuid, int64(app.UserPermEditOrg)}
	for _, field := range []struct{ name, column string }{
		{"org_uuid", "s.org_uuid"},
		{"social_post_uuid", "p.social_post_uuid"},
		{"social_post_target_uuid", "p.social_post_target_uuid"},
	} {
		if value, exists := gc.GetQuery(field.name); exists {
			if !app.ValidUUID(value) {
				apiRequest.Error(http.StatusBadRequest, field.name+" must be a valid UUID")
				return
			}
			args = append(args, value)
			query += fmt.Sprintf(" AND %s = $%d", field.column, len(args))
		}
	}
	if status, exists := gc.GetQuery("status"); exists {
		if status != "publishing" && status != "published" && status != "failed" && status != "uncertain" {
			apiRequest.Error(http.StatusBadRequest, "invalid publication status")
			return
		}
		args = append(args, status)
		query += fmt.Sprintf(" AND p.status = $%d", len(args))
	}
	query += " ORDER BY p.created_at, p.uuid"
	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	publications := make([]model.SocialPublication, 0)
	for rows.Next() {
		publication, err := scanSocialPublication(rows)
		if err != nil {
			apiRequest.InternalServerError()
			return
		}
		publications = append(publications, publication)
	}
	if rows.Err() != nil {
		apiRequest.InternalServerError()
		return
	}
	apiRequest.Success(http.StatusOK, gin.H{"publications": publications})
}
