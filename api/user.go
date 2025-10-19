package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func UserHandler(gc *gin.Context) {
	modeStr, _ := GetContextParam(gc, "mode")
	fmt.Println("query mode:", modeStr)

	switch modeStr {
	case "venues":
		QueryVenueForUser(gc)
		break

	case "venue-rights":
		QueryVenueRightsForUser(gc)
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}
