package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) AdminGetImageMeta(gc *gin.Context) {
	ctx := gc.Request.Context()
	// userId := gc.GetInt("user-id")

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	imageIndex, ok := ParamInt(gc, "imageIndex")
	if !ok || imageIndex < 1 || imageIndex > 4 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid image index"})
		return
	}

	// TODO: Permission Check!

	type ImageMeta struct {
		Id           *int           `json:"id"`
		FileName     *string        `json:"file_name"`
		Width        *int           `json:"width"`
		Height       *int           `json:"height"`
		MimeType     *string        `json:"mime_type"`
		AltText      *string        `json:"alt_text"`
		Description  *string        `json:"description"`
		LicenseID    *int           `json:"license_id"`
		Exif         map[string]any `json:"exif"`
		Expiration   *string        `json:"expiration_date"`
		CreatorName  *string        `json:"creator_name"`
		Copyright    *string        `json:"copyright"`
		FocusX       *float64       `json:"focus_x"`
		FocusY       *float64       `json:"focus_y"`
		MarginLeft   *int           `json:"margin_left"`
		MarginRight  *int           `json:"margin_right"`
		MarginTop    *int           `json:"margin_top"`
		MarginBottom *int           `json:"margin_bottom"`
	}
	imageCol := fmt.Sprintf("e.image%d_id", imageIndex)
	query := fmt.Sprintf(`
        SELECT
            p.id,
            p.file_name,
            p.width,
            p.height,
            p.mime_type,
            p.alt_text,
            p.description,
            p.license_id,
            p.exif,
            p.expiration_date,
            p.creator_name,
            p.copyright,
            p.focus_x,
            p.focus_y,
            p.margin_left,
            p.margin_right,
            p.margin_top,
            p.margin_bottom
        FROM %[1]s.event e
        LEFT JOIN %[1]s.pluto_image p
            ON p.id = %s
        WHERE e.id = $1
    `, h.DbSchema, imageCol)

	var meta ImageMeta
	err := h.DbPool.QueryRow(ctx, query, eventId).Scan(
		&meta.Id,
		&meta.FileName,
		&meta.Width,
		&meta.Height,
		&meta.MimeType,
		&meta.AltText,
		&meta.Description,
		&meta.LicenseID,
		&meta.Exif,
		&meta.Expiration,
		&meta.CreatorName,
		&meta.Copyright,
		&meta.FocusX,
		&meta.FocusY,
		&meta.MarginLeft,
		&meta.MarginRight,
		&meta.MarginTop,
		&meta.MarginBottom,
	)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, meta)
}
