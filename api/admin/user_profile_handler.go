package api_admin

import (
	"fmt"
	"image"
	"image/draw"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
)

func UserProfileHandler(gc *gin.Context) {
	// Todo: Return User Profile
	/*
		db := app.Singleton.MainDbPool
		ctx := gc.Request.Context()

		userId, ok := app.GetCurrentUserOrAbort(gc)
		if !ok {
			return // already sent error response
		}

	*/
}

func UserProfileUpdateHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool

	userId := api.UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	displayName := gc.PostForm("display_name")
	firstName := gc.PostForm("first_name")
	lastName := gc.PostForm("last_name")
	emailAddr := gc.PostForm("email")
	localeStr := gc.PostForm("locale")
	themeName := gc.PostForm("theme")

	// TODO: Validate email address

	// Begin DB transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	// Update existing user record
	sql := strings.Replace(`
        UPDATE {{schema}}.user
        SET display_name = $1,
            first_name = $2,
            last_name = $3,
            email_address = $4,
            locale = $5,
            theme = $6
        WHERE id = $7
    `, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	_, err = tx.Exec(
		ctx,
		sql,
		displayName,
		firstName,
		lastName,
		emailAddr,
		localeStr,
		themeName,
		userId,
	)
	if err != nil {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("update user failed: %v", err)})
		return
	}

	// Commit transaction
	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	saveDir := app.Singleton.Config.ProfileImageDir
	info, err := os.Stat(saveDir)
	if err != nil || !info.IsDir() {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "image directory does not exist"})
		return
	}

	file, err := gc.FormFile("avatar")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid image"})
		return
	}

	err = processImageAndSave(img, saveDir, userId, app.Singleton.Config.ProfileImageQuality)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process and save image"})
		return
	}

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
