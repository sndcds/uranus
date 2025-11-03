package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminDeleteUserAvatar(gc *gin.Context) {
	userId := gc.GetInt("user-id")

	profileImageDir := h.Config.ProfileImageDir
	info, err := os.Stat(profileImageDir)
	if err != nil || !info.IsDir() {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "profile image directory does not exist"})
		return
	}

	// File naming pattern: profile_img_<userId>_<size>.webp
	pattern := filepath.Join(profileImageDir, fmt.Sprintf("profile_img_%d_*.webp", userId))

	// Find all files that match the pattern
	files, err := filepath.Glob(pattern)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to search for avatar files: %v", err)})
		return
	}

	if len(files) == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"message": "no avatar images found for user"})
		return
	}

	var deletedFiles []string
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("failed to delete file %s: %v", filepath.Base(f), err),
			})
			return
		}
		deletedFiles = append(deletedFiles, filepath.Base(f))
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "avatar images deleted successfully",
	})
}
