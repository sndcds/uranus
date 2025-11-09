package api

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/sndcds/pluto"
)

type exifWalker struct {
	m map[string]string
}

func (w *exifWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	w.m[string(name)] = tag.String()
	return nil
}

func (h *ApiHandler) AdminUpdateEventImage(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema
	userId := gc.GetInt("user-id")

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	altText := gc.PostForm("alt_text")
	copyright := gc.PostForm("copyright")
	createdBy := gc.PostForm("created_by")

	licenseStr := gc.PostForm("license_id")
	fmt.Println("licenseStr:", licenseStr)

	var licenseId *int
	if licenseStr != "" {
		id, err := strconv.Atoi(licenseStr)
		if err != nil {
			gc.String(http.StatusBadRequest, "Invalid license_id")
			return
		}
		if id != 0 {
			licenseId = &id // only set pointer if not 0
		}
	}

	// Begin DB transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	file, err := gc.FormFile("image")
	if file != nil {
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
		cfg, _, err := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
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

		sql := strings.Replace(`
			INSERT INTO {{schema}}.pluto_image (file_name, gen_file_name, width, height, mime_type, exif, alt_text, created_by, copyright, license_id, user_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`, "{{schema}}", dbSchema, 1)
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
			licenseId,
			userId).Scan(&plutoImageId)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert pluto image failed: %v", err)})
			return
		}

		sql = strings.Replace(
			`DELETE FROM {{schema}}.event_image_links WHERE event_id = $1 AND main_image = TRUE`,
			"{{schema}}", dbSchema, 1)
		_, err = tx.Exec(ctx, sql, eventId)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert event image failed: %v", err)})
			return
		}

		sql = strings.Replace(
			`INSERT INTO {{schema}}.event_image_links (event_id, pluto_image_id, main_image, sort_index)
			VALUES ($1, $2, $3, $4)`,
			"{{schema}}", dbSchema, 1)
		_, err = tx.Exec(ctx, sql, eventId, plutoImageId, true, 0)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert event image failed: %v", err)})
			return
		}
	} else {
		sql := strings.Replace(
			`SELECT pluto_image_id FROM {{schema}}.event_image_links WHERE event_id = $1 AND main_image = TRUE`,
			"{{schema}}", dbSchema, 1)
		var plutoImageID string
		err := tx.QueryRow(ctx, sql, eventId).Scan(&plutoImageID)
		if err != nil {
			if err == pgx.ErrNoRows {
				gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("No main image found for event: %v", err)})
				return
			} else {
				gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		fmt.Println("Pluto Image ID:", plutoImageID)

		sql = fmt.Sprintf(`
			UPDATE %s.pluto_image
			SET alt_text = $1, copyright = $2, created_by = $3, license_id = $4
			WHERE id = $5`,
			dbSchema)

		// Execute the update
		_, err = tx.Exec(ctx, sql, altText, copyright, createdBy, licenseId, plutoImageID)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "image updated successfully"})
}
