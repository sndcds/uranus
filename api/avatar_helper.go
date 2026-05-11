package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) getAvatarURL(userUuid string) *string {
	imageDir := app.UranusInstance.Config.ProfileImageDir
	imagePath := filepath.Join(imageDir, fmt.Sprintf("profile_img_%s_64.webp", userUuid))

	if _, err := os.Stat(imagePath); err != nil {
		return nil
	}

	url := fmt.Sprintf("%s/api/user/%s/avatar/64", h.Config.BaseApiUrl, userUuid)
	return &url
}
