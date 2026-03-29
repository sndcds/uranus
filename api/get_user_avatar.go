package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetUserAvatar(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-user-avatar")
	size := ParamIntDefault(gc, "size", 256)

	userUuid := gc.Param("userUuid")
	if userUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "invalid userUuid")
		return
	}

	if size != 64 && size != 128 && size != 256 && size != 512 {
		apiRequest.Error(http.StatusBadRequest, "invalid image size (must be 64, 128, 256, or 512)")
		return
	}

	imageDir := app.UranusInstance.Config.ProfileImageDir
	imagePath := filepath.Join(imageDir, fmt.Sprintf("profile_img_%s_%d.webp", userUuid, size))

	file, err := os.Open(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			apiRequest.Error(http.StatusNotFound, "image not found")
		} else {
			apiRequest.Error(http.StatusInternalServerError, "failed to open image")
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
