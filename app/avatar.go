package app

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetAvatarURL(baseApiURL, imageDir, userUuid string, size int) *string {
	allowedSizes := map[int]bool{
		16: true, 32: true, 64: true, 128: true, 256: true, 512: true,
	}

	// Validate size
	if !allowedSizes[size] {
		return nil
	}

	// Build file path
	avatarFilePath := filepath.Join(
		imageDir,
		fmt.Sprintf("profile_img_%s_%d.webp", userUuid, size),
	)

	// Check if file exists
	if _, err := os.Stat(avatarFilePath); err != nil {
		return nil
	}

	// Build URL
	url := fmt.Sprintf("%s/api/user/%s/avatar/%d", baseApiURL, userUuid, size)
	return &url
}
