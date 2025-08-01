package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// getParam attempts to retrieve a parameter value from the Gin context.
//
// It first checks for a query parameter with the given name (e.g., /endpoint?name=value).
// If the query parameter is not present, it then checks the POST form body for the same parameter.
// If neither is found, or if the form parameter is an empty string, it returns false.
//
// Parameters:
//   - gc: the *gin.Context containing the HTTP request context.
//   - name: the name of the parameter to retrieve.
//
// Returns:
//   - string: the value of the parameter, if found.
//   - bool: true if the parameter was found in either query or form data and is non-empty; false otherwise.
//
// Example usage:
//
//	if val, ok := getParam(c, "user_id"); ok {
//	    // use val
//	} else {
//	    // handle missing parameter
//	}
func getParam(gc *gin.Context, name string) (string, bool) {
	val, exists := gc.GetQuery(name)
	if exists {
		return val, true
	}
	val = gc.PostForm(name)
	if val != "" {
		return val, true
	}
	return "", false
}

func QueryHandler(gc *gin.Context) {
	modeStr, _ := getParam(gc, "mode")
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

	case "user-venues":
		QueryVenueForUser(gc)
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}
