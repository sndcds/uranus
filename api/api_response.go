package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIResponse[T any] struct {
	Service      string                 `json:"service"`
	APIVersion   string                 `json:"api_version"`
	ResponseType string                 `json:"response_type"`
	Status       string                 `json:"status"`
	Timestamp    string                 `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Data         T                      `json:"data,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

func JSONSuccess[T any](gc *gin.Context, responseType string, data T, metadata map[string]interface{}) {
	resp := APIResponse[T]{
		Service:      "UranusAPI",
		APIVersion:   "1.0",
		ResponseType: responseType,
		Status:       "ok",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Metadata:     metadata,
		Data:         data,
	}
	gc.JSON(200, resp)
}

func JSONSuccessInfo(gc *gin.Context, responseType string) {
	resp := APIResponse[any]{ // 'any' because there is no Data
		Service:      "UranusAPI",
		APIVersion:   "1.0",
		ResponseType: responseType,
		Status:       "ok",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	gc.JSON(200, resp)
}

func JSONError(gc *gin.Context, responseType string, statusCode int, errMsg string) {
	resp := APIResponse[any]{ // 'any' because there is no Data
		Service:      "UranusAPI",
		APIVersion:   "1.0",
		ResponseType: responseType,
		Status:       "error",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Error:        errMsg,
	}
	gc.JSON(statusCode, resp)
}

func JSONDatabaseError(gc *gin.Context, responseType string) {
	JSONError(gc, responseType, http.StatusInternalServerError, "database error")
}
