package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) UpdateSpaceFields(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "admin-update-space-fields")

	spaceId, ok := ParamInt(gc, "spaceId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "spaceId is required")
		return
	}
	apiRequest.SetMeta("space_id", spaceId)

	var payload struct {
		Name                  NullableField[string]  `json:"name"`
		Description           NullableField[string]  `json:"description"`
		TotalCapacity         NullableField[int]     `json:"total_capacity"`
		SeatingCapacity       NullableField[int]     `json:"seating_capacity"`
		SpaceType             NullableField[string]  `json:"space_type"`
		BuildingLevel         NullableField[int]     `json:"building_level"`
		WebsiteLink           NullableField[string]  `json:"website_link"`
		AccessibilitySummary  NullableField[string]  `json:"accessibility_summary"`
		AccessibilityFlags    NullableField[string]  `json:"accessibility_flags"`
		AreaSqm               NullableField[float64] `json:"area_sqm"`
		EnvironmentalFeatures NullableField[int64]   `json:"environmental_features"`
		AudioFeatures         NullableField[int64]   `json:"audio_features"`
		PresentationFeatures  NullableField[int64]   `json:"presentation_features"`
		LightingFeatures      NullableField[int64]   `json:"lighting_features"`
		ClimateFeatures       NullableField[int64]   `json:"climate_features"`
		MiscFeatures          NullableField[int64]   `json:"misc_features"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	argPos = addUpdateClauseNullable("name", payload.Name, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("description", payload.Description, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("total_capacity", payload.TotalCapacity, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("seating_capacity", payload.SeatingCapacity, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("space_type", payload.SpaceType, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("building_level", payload.BuildingLevel, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("website_link", payload.WebsiteLink, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("accessibility_summary", payload.AccessibilitySummary, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("accessibility_flags", payload.AccessibilityFlags, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("area_sqm", payload.AreaSqm, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("environmental_features", payload.EnvironmentalFeatures, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("audio_features", payload.AudioFeatures, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("presentation_features", payload.PresentationFeatures, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("lighting_features", payload.LightingFeatures, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("climate_features", payload.ClimateFeatures, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("misc_features", payload.MiscFeatures, &setClauses, &args, argPos)

	if len(setClauses) == 0 {
		apiRequest.SuccessNoData(http.StatusOK, "no fields updated")
		return
	}
	apiRequest.SetMeta("field_count", len(setClauses))

	query := fmt.Sprintf(`UPDATE %s.space SET %s WHERE id = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)

	args = append(args, spaceId) // eventId is the last parameter

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		res, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to update event: %v", err),
			}
		}

		if res.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("event not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "space", []int{spaceId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed"),
			}
		}

		return nil
	})
	if txErr != nil {
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "space fields updated")
}
