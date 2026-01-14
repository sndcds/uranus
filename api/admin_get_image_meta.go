package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/model"
)

// TODO: Review code

func (h *ApiHandler) AdminGetImageMeta(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	imageIndex, ok := ParamInt(gc, "imageIndex")
	if !ok || imageIndex < 1 || imageIndex > 4 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid image index"})
		return
	}

	// TODO: Permission Check!

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

	var meta model.ImageMeta
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
