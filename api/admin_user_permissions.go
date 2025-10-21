package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"net/http"
)

func AdminUserPermissionsHandler(gc *gin.Context) {
	modeStr := gc.Param("mode")

	switch modeStr {
	case "all":
		fetchUserPermissions(gc)
		break
	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}

func fetchUserPermissions(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	userId, ok := app.GetCurrentUserOrAbort(gc)
	if !ok {
		return // already sent error response
	}

	sql := app.Singleton.SqlAdminUserPermissions
	rows, err := db.Query(ctx, sql, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = string(fd.Name)
	}

	var permission []map[string]interface{}

	for rows.Next() {
		// Create a slice of empty interfaces for Scan
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// Scan into []interface{}
		if err := rows.Scan(valuePtrs...); err != nil {
			gc.JSON(http.StatusInternalServerError, err)
			return
		}

		// Build a map for this row
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for text fields
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}

		permission = append(permission, rowMap)
	}

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, rows.Err())
		return
	}

	// Wrap in outer object with metadata
	response := gin.H{
		"api":         "Uranus",
		"version":     "1.0.0",
		"language":    "en",
		"permissions": permission,
	}

	gc.JSON(http.StatusOK, response)
}
