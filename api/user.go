package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func UserHandler(gc *gin.Context) {
	modeStr, _ := getParam(gc, "mode")
	fmt.Println("query mode:", modeStr)

	switch modeStr {
	case "venues":
		QueryVenueForUser(gc)
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}
