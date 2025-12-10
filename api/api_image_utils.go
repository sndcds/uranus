package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: Review code

// GetEventImageField returns the event table column name for a given imageIndex (1-8)
func GetEventImageField(imageIndex int) (string, error) {
	columns := map[int]string{
		1: "image1_id",
		2: "image2_id",
		3: "image3_id",
		4: "image4_id",
		5: "image_some_16_9_id",
		6: "image_some_4_5_id",
		7: "image_some_9_16_id",
		8: "image_some_1_1_id",
	}

	col, ok := columns[imageIndex]
	if !ok {
		return "", fmt.Errorf("Invalid imageIndex %d: must be 1-8", imageIndex)
	}
	return col, nil
}

func (h *ApiHandler) updateEventImage(
	gc *gin.Context, tx pgx.Tx, eventId int, imageIndex int, imageId int) error {
	ctx := gc.Request.Context()

	fieldName, err := GetEventImageField(imageIndex)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
        UPDATE %s.event SET %s = $1, modified_at = CURRENT_TIMESTAMP WHERE id = $2`,
		h.DbSchema, fieldName)
	_, err = tx.Exec(ctx, query, imageId, eventId)
	if err != nil {
		return fmt.Errorf("Failed to update image")
	}

	return nil
}

// GetEventImageId fetches the image ID for a given event and imageIndex
func (h *ApiHandler) GetEventImageId(
	gc *gin.Context, tx pgx.Tx, eventId int, imageIndex int) (int, error) {
	ctx := gc.Request.Context()

	fieldName, err := GetEventImageField(imageIndex)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`SELECT %s FROM %s.event WHERE id = $1`, fieldName, h.DbSchema)
	var imageId *int
	err = tx.QueryRow(ctx, query, eventId).Scan(&imageId)
	if err != nil {
		return 0, fmt.Errorf("Failed to fetch image ID")
	}

	if imageId == nil {
		return -1, nil // no image set
	}

	return *imageId, nil
}
