package api

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

type exifWalker struct {
	m map[string]string
}

func (w *exifWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	w.m[string(name)] = tag.String()
	return nil
}

func (h *ApiHandler) AdminUpsertEventImage(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

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

	focusX, err := getPostFormFloatPtr(gc, "focus_x")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "focus_x must be a float number"})
		return
	}

	focusY, err := getPostFormFloatPtr(gc, "focus_y")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "focus_y must be a float number"})
		return
	}

	licenseId, err := getPostFormIntPtr(gc, "license_id")
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid license_id"})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		file, err := gc.FormFile("image")
		if file != nil { // Upload a new file
			// Read file into buffer for multiple uses
			buf := new(bytes.Buffer)
			src, err := file.Open()
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to open uploaded file: %v", err),
				}
			}
			defer src.Close()

			if _, err := io.Copy(buf, src); err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to read uploaded file: %v", err),
				}
			}

			// Detect MIME type (use only first 512 bytes for detection)
			mimeType := http.DetectContentType(buf.Bytes()[:512])
			fmt.Println("MIME type:", mimeType)

			// Decode image config for dimensions
			cfg, _, err := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
			if err != nil {
				return &ApiTxError{
					Code: http.StatusBadRequest,
					Err:  fmt.Errorf("invalid image: %v", err),
				}
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
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to generate filename: %v", err),
				}
			}

			// Ensure upload directory exists
			saveDir := h.Config.PlutoImageDir
			if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to create directory: %v", err),
				}
			}

			generatedFileName = fmt.Sprintf("event_%d_%s", eventId, generatedFileName)
			savePath := filepath.Join(saveDir, generatedFileName)
			fmt.Println(savePath)
			if err = os.WriteFile(savePath, buf.Bytes(), 0644); err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to save file: %v", err),
				}
			}

			plutoRemoveImageId, err = h.GetEventImageId(gc, tx, eventId, imageIndex)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to get previous image Id"),
				}
			}

			err = tx.QueryRow(
				ctx,
				app.Singleton.SqlInsertPlutoImage,
				originalFileName, generatedFileName,
				cfg.Width, cfg.Height, mimeType, exifData,
				altText, copyright, creatorName, licenseId, description,
				focusX, focusY,
				userId).Scan(&plutoImageId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert pluto image: %v", err),
				}
			}

			err = h.updateEventImage(gc, tx, eventId, imageIndex, plutoImageId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  err,
				}
			}
		} else { // Update meta
			var plutoImageIdErr error
			plutoImageId, plutoImageIdErr = h.GetEventImageId(gc, tx, eventId, imageIndex)
			if plutoImageIdErr != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("No main image found for event"),
				}
			}

			_, err = tx.Exec(
				ctx,
				app.Singleton.SqlUpdatePlutoImageMeta,
				altText,
				copyright,
				creatorName,
				licenseId,
				description,
				focusX, focusY,
				plutoImageId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  err,
				}
			}
		}

		if plutoRemoveImageId >= 0 {
			// Delete all pluto_cache entries for 'plutoRemoveImageId'
			query := fmt.Sprintf(`DELETE FROM %s.pluto_cache WHERE image_id = $1`, h.DbSchema)
			cmdTag, err := tx.Exec(ctx, query, plutoRemoveImageId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusBadRequest,
					Err:  fmt.Errorf("failed to delete Pluto cache"),
				}
			}
			plutoDeleteCacheCount = cmdTag.RowsAffected()

			// Delete pluto_image for 'plutoRemoveImageId'
			query = fmt.Sprintf(`DELETE FROM %s.pluto_image WHERE id = $1 RETURNING gen_file_name`, h.DbSchema)
			row := tx.QueryRow(ctx, query, plutoRemoveImageId)
			err = row.Scan(&plutoPrevFileName)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusBadRequest,
					Err:  fmt.Errorf("failed to delete Pluto image"),
				}
			}
		}

		// Delete all related files
		if plutoRemoveImageId >= 0 {
			// TODO: Log errors
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

		err = RefreshEventProjections(ctx, tx, "event", []int{eventId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})

	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
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
