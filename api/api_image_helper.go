package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
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
	plutoImageUuid string,
	context string,
	contextUuid string,
	identifier string,
) error {
	ctx := gc.Request.Context()

	query := fmt.Sprintf(`
		INSERT INTO %s.pluto_image_link
			(pluto_image_uuid, context, context_uuid, identifier)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (context, context_uuid, identifier)
		DO UPDATE SET
			pluto_image_uuid = EXCLUDED.pluto_image_uuid
	`, h.DbSchema)

	_, err := h.DbPool.Exec(
		ctx,
		query,
		plutoImageUuid,
		context,
		contextUuid,
		identifier,
	)

	if err != nil {
		return err
	}

	return nil
}

func ImageUrl(imageUuid string) string {
	return fmt.Sprintf(
		"%s/api/image/%s",
		app.UranusInstance.Config.BaseApiUrl,
		imageUuid,
	)
}
