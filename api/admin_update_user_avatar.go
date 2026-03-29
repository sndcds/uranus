package api

import (
	"fmt"
	"image"
	"image/draw"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminUploadUserAvatar(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "ulpoad-user-avatar")
	userUuid := h.userUuid(gc)

	profileImageDir := h.Config.ProfileImageDir
	info, err := os.Stat(profileImageDir)
	if err != nil || !info.IsDir() {
		apiRequest.Error(http.StatusInternalServerError, "image directory does not exist")
		return
	}

	file, err := gc.FormFile("avatar")
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "avatar image file is required")
		return
	}

	src, err := file.Open()
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to open uploaded file")
		return
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "invalid image file format")
		return
	}

	err = processImageAndSave(img, profileImageDir, userUuid, h.Config.ProfileImageQuality)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusInternalServerError, "failed to process and save image")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "profile image saved successfully")
}

func processImageAndSave(img image.Image, saveDir string, userUuid string, quality float32) error {
	// Get width and height
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Determine max side and compute cropping rectangle (center crop)
	var cropRect image.Rectangle
	if w > h {
		offset := (w - h) / 2
		cropRect = image.Rect(offset, 0, offset+h, h)
	} else {
		offset := (h - w) / 2
		cropRect = image.Rect(0, offset, w, offset+w)
	}

	// Crop to square
	squareImg := image.NewRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
	draw.Draw(squareImg, squareImg.Bounds(), img, cropRect.Min, draw.Src)

	// Sizes you want to save (in pixels)
	sizes := []int{512, 256, 128, 64}

	// Loop through and save each version
	for _, size := range sizes {
		resized := resize.Resize(uint(size), uint(size), squareImg, resize.Lanczos3)

		// Example filename: profile_img_123_256.webp
		savePath := filepath.Join(saveDir, fmt.Sprintf("profile_img_%s_%d.webp", userUuid, size))

		outFile, err := os.Create(savePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", savePath, err)
		}

		// Use lossy compression, quality
		err = webp.Encode(outFile, resized, &webp.Options{Lossless: false, Quality: quality})
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to encode %s: %v", savePath, err)
		}
	}

	return nil
}
