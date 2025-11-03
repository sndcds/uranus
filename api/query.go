package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func QueryHandler(gc *gin.Context) {
	modeStr, _ := GetContextParam(gc, "mode")

	switch modeStr {
	case "venue":
		// QueryVenue(gc)
		break

	case "space":
		// QuerySpace(gc)
		break

	case "organizer":
		break

	case "user-venues":
		QueryVenueForUser(gc) // TODO: Auslagern!!!!!!
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}
