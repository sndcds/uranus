package api

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/sndcds/pluto"
)

func (h *ApiHandler) AdminUpdateEventTeaserImage(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	userId := gc.GetInt("user-id")

	altText := gc.PostForm("alt_text")
	copyright := gc.PostForm("copyright")
	createdBy := gc.PostForm("created_by")
	licenseId := gc.PostForm("license_id") // TODO: Handle string to int as in AdminUpdateEventImage

	// Handle file upload
	file, err := gc.FormFile("image")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("image file is required: %v", err)})
		return
	}

	// Read file into buffer for multiple uses
	buf := new(bytes.Buffer)
	src, err := file.Open()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to open uploaded file: %v", err)})
		return
	}
	defer src.Close()

	if _, err := io.Copy(buf, src); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read uploaded file: %v", err)})
		return
	}

	// Detect MIME type (use only first 512 bytes for detection)
	mimeType := http.DetectContentType(buf.Bytes()[:512])
	fmt.Println("MIME type:", mimeType)

	// Decode image config for dimensions
	cfg, format, err := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid image: %v", err)})
		return
	}

	// Decode EXIF metadata if present
	exifData := make(map[string]string)
	if x, err := exif.Decode(bytes.NewReader(buf.Bytes())); err == nil {
		x.Walk(&exifWalker{m: exifData})
	}

	// Sanitize and generate filename
	originalFileName := filepath.Base(file.Filename)
	generatedFileName, err := pluto.GenerateImageFilename(originalFileName)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate filename: %v", err)})
		return
	}

	// Ensure upload directory exists
	saveDir := h.Config.PlutoImageDir
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create directory: %v", err)})
		return
	}

	generatedFileName = fmt.Sprintf("event_%s_%s", eventId, generatedFileName)
	savePath := filepath.Join(saveDir, generatedFileName)
	fmt.Println(savePath)
	if err := os.WriteFile(savePath, buf.Bytes(), 0644); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file: %v", err)})
		return
	}

	// Begin DB transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sql := strings.Replace(`
        INSERT INTO {{schema}}.pluto_image (file_name, gen_file_name, width, height, mime_type, exif, alt_text, created_by, copyright, user_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`, "{{schema}}", dbSchema, 1)
	var plutoImageId int64
	err = tx.QueryRow(
		ctx,
		sql,
		originalFileName,
		generatedFileName,
		cfg.Width, cfg.Height,
		mimeType,
		exifData,
		altText,
		createdBy,
		copyright,
		userId).Scan(&plutoImageId)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert pluto image failed: %v", err)})
		return
	}

	sql = strings.Replace(`INSERT INTO {{schema}}.event_image_link (event_id, pluto_image_id, main_image, sort_index)
	VALUES ($1, $2, $3, $4)`, "{{schema}}", dbSchema, 1)

	_, err = tx.Exec(ctx, sql, eventId, plutoImageId, true, 0)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert event image failed: %v", err)})
		return
	}

	// Commit transaction
	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	// Return success
	gc.JSON(http.StatusOK, gin.H{
		"message":           "image updated successfully",
		"filename":          generatedFileName,
		"original_filename": originalFileName,
		"width":             cfg.Width,
		"height":            cfg.Height,
		"format":            format,
		"exif":              exifData,
		"alt_text":          altText,
		"copyright":         copyright,
		"created_by":        createdBy,
		"license_id":        licenseId,
	})
}
