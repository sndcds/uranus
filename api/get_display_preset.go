package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) GetDisplayPreset(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-display-preset")
	ctx := gc.Request.Context()

	uuid := gc.Param("uuid")
	code := gc.Query("code")
	if len(code) < 4 {
		apiRequest.Error(http.StatusBadRequest, "code must be at least 4 characters")
		return
	}

	query := fmt.Sprintf(`
		SELECT uuid, org_uuid, name, description, display_mode, options
		FROM %s.display_preset
		WHERE uuid = $1::uuid AND code = $2
		`,
		h.DbSchema)

	var preset model.DisplayPreset
	err := h.DbPool.QueryRow(ctx, query, uuid, code).Scan(
		&preset.Uuid,
		&preset.OrgUuid,
		&preset.Name,
		&preset.Description,
		&preset.DisplayMode,
		&preset.Options,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apiRequest.Error(http.StatusNotFound, "display preset not found")
			return
		}

		apiRequest.Error(http.StatusInternalServerError, "failed to load display preset")
		return
	}

	apiRequest.Success(http.StatusOK, preset, "display preset loaded successfully")
}
