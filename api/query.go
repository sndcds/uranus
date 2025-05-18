package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func getParam(c *gin.Context, name string) string {
	return c.DefaultQuery(name, c.PostForm(name))
}

func QueryHandler(c *gin.Context) {

	modeStr := getParam(c, "mode")
	fmt.Println("query mode:", modeStr)

	switch modeStr {
	case "event":
		QueryEvent(c)
		break

	case "venue":
		// QueryVenue(c)
		break

	case "space":
		// QuerySpace(c)
		break

	case "organization":
		// QueryOrganization(c)
		break

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}

}
