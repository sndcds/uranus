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
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUploadUserAvatar(gc *gin.Context) {
	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	profileImageDir := app.Singleton.Config.ProfileImageDir
	fmt.Println("profileImageDir", profileImageDir)
	info, err := os.Stat(profileImageDir)
	if err != nil || !info.IsDir() {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "image directory does not exist"})
		return
	}

	fmt.Println("1")
	file, err := gc.FormFile("avatar")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}

	fmt.Println("2")
	src, err := file.Open()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer src.Close()

	fmt.Println("3")
	img, _, err := image.Decode(src)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid image"})
		return
	}

	fmt.Println("4")
	err = processImageAndSave(img, profileImageDir, userId, app.Singleton.Config.ProfileImageQuality)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process and save image"})
		return
	}

	fmt.Println("5")
	gc.JSON(http.StatusOK, gin.H{
		"message": "profile image saved successfully",
	})
}

func processImageAndSave(img image.Image, saveDir string, userId int, quality float32) error {
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
		savePath := filepath.Join(saveDir, fmt.Sprintf("profile_img_%d_%d.webp", userId, size))

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
