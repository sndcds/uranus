package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) AdminUserPermissions(gc *gin.Context) {
	db := h.DbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	sql := app.Singleton.SqlAdminUserPermissions

	rows, err := db.Query(ctx, sql, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = string(fd.Name)
	}

	var permissions []map[string]interface{}

	for rows.Next() {
		// Scan row
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				val = string(b)
			}

			// Rename columns
			switch col {
			case "id":
				rowMap["role_id"] = val
			case "name":
				rowMap["role_name"] = val
			default:
				rowMap[col] = val
			}
		}

		permissions = append(permissions, rowMap)
	}

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": rows.Err().Error()})
		return
	}

	if len(permissions) == 0 {
		gc.JSON(http.StatusOK, gin.H{"message": "user has no permissions"})
		return
	}

	gc.JSON(http.StatusOK, permissions)
}
