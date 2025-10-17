package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func QueryHandler(gc *gin.Context) {
	modeStr, _ := GetContextParam(gc, "mode")

	switch modeStr {
	case "event":
		QueryEvent(gc)
		break

	case "venue":
		// QueryVenue(gc)
		break

	case "venue-map":
		QueryVenueForMap(gc)
		break

	case "space":
		// QuerySpace(gc)
		break

	case "organizer":
		break

	case "user-venues":
		QueryVenueForUser(gc)
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}
