package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetVenueUuidBySlug(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-venue-slug")
	ctx := gc.Request.Context()

	slug := gc.Param("slug")
	if slug == "" {
		apiRequest.Required("slug is required")
		return
	}

	query := fmt.Sprintf("SELECT uuid FROM %s.venue WHERE slug = $1", h.DbSchema)

	var uuid *string
	err := h.DbPool.QueryRow(ctx, query, slug).Scan(&uuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.SetMeta("err_code", "1001")
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, gin.H{
		"uuid": uuid,
		"slug": slug,
	})
}
