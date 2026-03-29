package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteUserAvatar(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "delete-user-avatar")
	userUuid := h.userUuid(gc)

	profileImageDir := h.Config.ProfileImageDir
	info, err := os.Stat(profileImageDir)
	if err != nil || !info.IsDir() {
		debugf("%s is not a directory", profileImageDir)
		apiRequest.Error(http.StatusInternalServerError, "profile image directory does not exist")
		return
	}

	// File naming pattern: profile_img_<userUuid>_<size>.webp
	pattern := filepath.Join(profileImageDir, fmt.Sprintf("profile_img_%s_*.webp", userUuid))

	// Find all files that match the pattern
	files, err := filepath.Glob(pattern)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusInternalServerError, "failed to search for avatar files")
		return
	}

	if len(files) == 0 {
		apiRequest.Error(http.StatusNotFound, "no avatar images found for user")
		return
	}

	var deletedFiles []string
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			debugf("failed to delete file %s: %v", filepath.Base(f), err)
			apiRequest.Error(http.StatusInternalServerError, "failed to delete file")
			return
		}
		deletedFiles = append(deletedFiles, filepath.Base(f))
	}

	apiRequest.SuccessNoData(http.StatusOK, "avatar images deleted successfully")
}
