package api_admin

import (
	"fmt"
	"image"
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
	langStr := gc.PostForm("lang")
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
		langStr,
		themeName,
		userId)
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

	// Downscale if larger than 512x512
	maxSize := uint(512)
	w := uint(img.Bounds().Dx())
	h := uint(img.Bounds().Dy())
	if w > maxSize || h > maxSize {
		img = resize.Thumbnail(maxSize, maxSize, img, resize.Lanczos3)
	}

	// Save as WebP
	savePath := filepath.Join(saveDir, fmt.Sprintf("profile_img_%d.webp", userId))
	outFile, err := os.Create(savePath)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save image"})
		return
	}
	defer outFile.Close()

	if err := webp.Encode(outFile, img, &webp.Options{Lossless: false, Quality: 80}); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode webp"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "profile image saved successfully",
	})
}
