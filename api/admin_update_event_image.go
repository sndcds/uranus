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
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/app"
)

type exifWalker struct {
	m map[string]string
}

func (w *exifWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	w.m[string(name)] = tag.String()
	return nil
}

func (h *ApiHandler) AdminUpsertEventImage(gc *gin.Context) {
	h.InitFromGin(gc)

	plutoImageId := -1
	plutoRemoveImageId := -1
	plutoDeleteCacheCount := int64(0)
	plutoRemovedCacheFileCount := 0
	plutoPrevFileName := ""

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
		return
	}

	imageIndex, ok := ParamInt(gc, "imageIndex")
	if !ok || imageIndex < 1 || imageIndex > 8 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid imageIndex"})
		return
	}

	altText := getPostFormPtr(gc, "alt_text")
	copyright := getPostFormPtr(gc, "copyright")
	creatorName := getPostFormPtr(gc, "creator_name")
	description := getPostFormPtr(gc, "description")

	licenseId, err := getPostFormIntPtr(gc, "license_id")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid license_id"})
		return
	}

	// Begin DB transaction
	tx, err := h.DbPool.Begin(h.Context)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback(h.Context) }()

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
		x, err := exif.Decode(bytes.NewReader(buf.Bytes()))
		if err == nil {
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

		generatedFileName = fmt.Sprintf("event_%d_%s", eventId, generatedFileName)
		savePath := filepath.Join(saveDir, generatedFileName)
		fmt.Println(savePath)
		if err = os.WriteFile(savePath, buf.Bytes(), 0644); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file: %v", err)})
			return
		}

		plutoRemoveImageId, err = h.GetEventImageId(tx, eventId, imageIndex)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get previous image ID"})
			return
		}

		query := strings.Replace(`
			INSERT INTO {{schema}}.pluto_image (file_name, gen_file_name, width, height, mime_type, exif, alt_text, creator_name, copyright, license_id, user_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`, "{{schema}}", h.DbSchema, 1)
		err = tx.QueryRow(
			h.Context,
			query,
			originalFileName,
			generatedFileName,
			cfg.Width, cfg.Height,
			mimeType,
			exifData,
			altText,
			creatorName,
			copyright,
			licenseId,
			h.UserId).Scan(&plutoImageId)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "Insert pluto image failed"})
			return
		}

		err = h.updateEventImage(tx, eventId, imageIndex, plutoImageId)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		var plutoImageIdErr error
		plutoImageId, plutoImageIdErr = h.GetEventImageId(tx, eventId, imageIndex)
		if plutoImageIdErr != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "No main image found for event"})
			return
		}

		query := fmt.Sprintf(`
			UPDATE %s.pluto_image
			SET alt_text = $1, copyright = $2, creator_name = $3, license_id = $4, description = $5
			WHERE id = $6`,
			h.DbSchema)
		_, err = tx.Exec(h.Context, query, altText, copyright, creatorName, licenseId, description, plutoImageId)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if plutoRemoveImageId >= 0 {
		// Delete all pluto_cache entries for 'plutoRemoveImageId'
		query := fmt.Sprintf(`DELETE FROM %s.pluto_cache WHERE image_id = $1`, h.DbSchema)
		cmdTag, err := tx.Exec(h.Context, query, plutoRemoveImageId)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "failed to delete Pluto cache"})
			return
		}
		plutoDeleteCacheCount = cmdTag.RowsAffected()

		// Delete pluto_image for 'plutoRemoveImageId'
		query = fmt.Sprintf(`DELETE FROM %s.pluto_image WHERE id = $1 RETURNING gen_file_name`, h.DbSchema)
		row := tx.QueryRow(h.Context, query, plutoRemoveImageId)
		err = row.Scan(&plutoPrevFileName)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "failed to delete Pluto image"})
			return
		}
	}

	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	// Delete all related files
	if plutoRemoveImageId >= 0 {
		cacheFilePrefix := fmt.Sprintf("%x_", plutoRemoveImageId)
		fmt.Println("DeleteFilesWithPrefix:", cacheFilePrefix)
		plutoRemovedCacheFileCount, err = app.DeleteFilesWithPrefix(h.Config.PlutoCacheDir, cacheFilePrefix)
		if err != nil {
			// TODO: Log!
			fmt.Printf(err.Error())
		}

		filePath := h.Config.PlutoImageDir + "/" + plutoPrevFileName

		fmt.Println("RemoveFile:", filePath)
		err = app.RemoveFile(filePath)
		if err != nil {
			// TODO: Log!
			fmt.Printf(err.Error())
		}

		fmt.Printf("Deleted %d files in cache\n", plutoRemovedCacheFileCount)
		fmt.Println("Deleted file:", plutoPrevFileName)
	}

	// Build JSON response
	response := gin.H{
		"message": "image updated successfully",
	}

	if plutoImageId >= 0 {
		response["image_id"] = plutoImageId
	}
	if plutoRemoveImageId >= 0 {
		response["replaced_image_id"] = plutoRemoveImageId
	}
	if plutoDeleteCacheCount > 0 {
		response["cache_files"] = plutoDeleteCacheCount
		response["removed_cache_files"] = plutoRemovedCacheFileCount
	}

	gc.JSON(http.StatusOK, response)
}
