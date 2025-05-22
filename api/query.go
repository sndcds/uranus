package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func getParam(gc *gin.Context, name string) string {
	return gc.DefaultQuery(name, gc.PostForm(name))
}

func QueryHandler(gc *gin.Context) {
	modeStr := getParam(gc, "mode")
	fmt.Println("query mode:", modeStr)

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

	case "organization":
		// QueryOrganization(gc)
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}
