package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"net/http"
)

func EventImagesHandler(gc *gin.Context) {
	eventID, ok := ParamAsIntMessageOnFail(gc, "event-id")
	if !ok {
		return
	}

	fmt.Println("eventID", eventID)

	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	sql := app.Singleton.SqlEventImages
	rows, err := db.Query(ctx, sql, eventID)
	if InternalServerErrorAnswer(gc, err) {
		return
	}
	defer rows.Close()

	type EventImage struct {
		ID           *int    `json:"id"`
		PlutoImageID *int    `json:"pluto_image_id"`
		FileName     *string `json:"file_name"`
		Width        *int    `json:"width"`
		Height       *int    `json:"height"`
		MimeType     *string `json:"mime_type"`
		AltText      *string `json:"alt_text"`
		Caption      *string `json:"caption"`
		Copyright    *string `json:"copyright"`
		License      *string `json:"license"`
		FocusX       *int    `json:"focus_x"`
		FocusY       *int    `json:"focus_y"`
	}

	var images []EventImage
	for rows.Next() {
		var img EventImage
		err := rows.Scan(
			&img.ID,
			&img.PlutoImageID,
			&img.FileName,
			&img.Width,
			&img.Height,
			&img.MimeType,
			&img.AltText,
			&img.Caption,
			&img.Copyright,
			&img.License,
			&img.FocusX,
			&img.FocusY,
		)
		if InternalServerErrorAnswer(gc, err) {
			return
		}
		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		InternalServerErrorAnswer(gc, err)
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"api":     app.Singleton.APIName,
		"version": app.Singleton.APIVersion,
		"images":  images,
	})
}
