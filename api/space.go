package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func SpaceHandler(gc *gin.Context) {
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
