package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) GetSpace(gc *gin.Context) {
	modeStr, _ := GetContextParam(gc, "mode")

	switch modeStr {
	case "spaces-for-venue":
		QuerySpacesByVenue(gc)
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}
