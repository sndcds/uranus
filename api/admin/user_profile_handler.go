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
	dbSchema := app.Singleton.Config.DbSchema

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

	sql := strings.Replace(`
        INSERT INTO {{schema}}.user_profile (user_id, display_name, first_name, last_name, email, iso_639_1, theme)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`, "{{schema}}", dbSchema, 1)
	_, err = tx.Exec(
		ctx,
		sql,
		userId,
		displayName,
		firstName,
		lastName,
		emailAddr,
		localeStr,
		themeName)
	if err != nil {
		_ = tx.Rollback(gc)
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert user profile failed: %v", err)})
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

	if err := processImageAndSave(img, saveDir, userId); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process and save image"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "profile image saved successfully",
	})
}

func processImageAndSave(img image.Image, saveDir string, userId int) error {
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

	// Crop the image
	squareImg := image.NewRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
	draw.Draw(squareImg, squareImg.Bounds(), img, cropRect.Min, draw.Src)

	// Resize to 512x512
	resized := resize.Resize(512, 512, squareImg, resize.Lanczos3)

	// Save as WebP
	savePath := filepath.Join(saveDir, fmt.Sprintf("profile_img_%d.webp", userId))
	outFile, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return webp.Encode(outFile, resized, &webp.Options{Lossless: true})
}
