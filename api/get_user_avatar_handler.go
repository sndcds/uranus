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
	userIdStr := gc.Param("userId")
	sizeStr := gc.Param("size")

	fmt.Println(userIdStr)
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Default to 256px if no size provided
	size := 256
	fmt.Println("sizeStr", sizeStr)
	if sizeStr != "" {
		s, err := strconv.Atoi(sizeStr)
		if err != nil || (s != 64 && s != 128 && s != 256 && s != 512) {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid image size (must be 64, 128, 256, or 512)"})
			return
		}
		size = s
	}

	imageDir := app.Singleton.Config.ProfileImageDir
	imagePath := filepath.Join(imageDir, fmt.Sprintf("profile_img_%d_%d.webp", userId, size))

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
