package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func GetUserAvatarHandler(gc *gin.Context) {
	userIdStr := gc.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	profileImageDir := app.Singleton.Config.ProfileImageDir
	imagePath := filepath.Join(profileImageDir, fmt.Sprintf("profile_img_%d.webp", userId))

	file, err := os.Open(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open image"})
		}
		return
	}
	defer file.Close()

	gc.Header("Content-Type", "image/webp")

	if _, err := io.Copy(gc.Writer, file); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serve image"})
		return
	}
}
