package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func IsEventImageIdentifier(identifier string) bool {
	mapping := map[string]int{
		"main":      1,
		"gallery1":  2,
		"gallery2":  3,
		"gallery3":  4,
		"some_16_9": 5,
		"some_1_1":  6,
		"some_4_5":  7,
		"some_9_16": 8,
	}

	_, ok := mapping[identifier]
	return ok
}

func IsOrganizationImageIdentifier(identifier string) bool {
	mapping := map[string]int{
		"main_logo":        1,
		"dark_theme_logo":  2,
		"light_theme_logo": 3,
		"avatar":           4,
	}

	_, ok := mapping[identifier]
	return ok
}

func IsVenueImageIdentifier(identifier string) bool {
	mapping := map[string]int{
		"main_logo":        1,
		"dark_theme_logo":  2,
		"light_theme_logo": 3,
		"avatar":           4,
		"main_photo":       5,
		"gallery_photo_1":  6,
		"gallery_photo_2":  7,
		"gallery_photo_3":  8,
	}

	_, ok := mapping[identifier]
	return ok
}

func (h *ApiHandler) UpsertImage(
	gc *gin.Context,
	plutoImageId int,
	context string,
	contextId int,
	identifier string,
) (int, error) {
	ctx := gc.Request.Context()

	query := fmt.Sprintf(`
		INSERT INTO %s.pluto_image_link
			(pluto_image_id, context, context_id, identifier)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (context, context_id, identifier)
		DO UPDATE SET
			pluto_image_id = EXCLUDED.pluto_image_id
		RETURNING id
	`, h.DbSchema)

	var plutoImageLinkId int
	err := h.DbPool.QueryRow(
		ctx,
		query,
		plutoImageId,
		context,
		contextId,
		identifier,
	).Scan(&plutoImageLinkId)

	if err != nil {
		return 0, err
	}

	return plutoImageId, nil
}
