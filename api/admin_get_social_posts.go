package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetSocialPosts(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-social-posts")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)
	orgUuid, filtered := gc.GetQuery("org_uuid")
	if filtered && !app.ValidUUID(orgUuid) {
		apiRequest.Error(http.StatusBadRequest, "org_uuid must be a valid UUID")
		return
	}

	// Match the social-account list: organization filter and permission-scoped
	// results, without a separate pagination convention.
	query := h.socialPostQuery() + fmt.Sprintf(`
		WHERE EXISTS (SELECT 1 FROM %s.user_organization_link l
			JOIN %s.organization_member_link m
				ON m.org_uuid = l.org_uuid AND m.user_uuid = l.user_uuid AND m.has_joined = true
			WHERE l.org_uuid = p.org_uuid AND l.user_uuid = $1
			AND (l.permissions & $2) = $2)`, h.DbSchema, h.DbSchema)
	args := []any{userUuid, int64(app.UserPermEditOrg)}
	if filtered {
		query += " AND p.org_uuid = $3"
		args = append(args, orgUuid)
	}
	query += " ORDER BY p.created_at, p.uuid"
	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	posts := make([]model.SocialPost, 0)
	for rows.Next() {
		post, err := scanSocialPost(rows)
		if err != nil {
			apiRequest.InternalServerError()
			return
		}
		posts = append(posts, post)
	}
	if rows.Err() != nil {
		apiRequest.InternalServerError()
		return
	}
	apiRequest.Success(http.StatusOK, gin.H{"posts": posts})
}
