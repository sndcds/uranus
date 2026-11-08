package app

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// AllowedAuthOrigin checks whether an origin is in the configured allowlist.
func (app *Uranus) AllowedAuthOrigin(origin string) bool {
	for _, allowed := range app.Config.AuthAllowedOrigins {
		if origin == allowed {
			return true
		}
	}

	return false
}

func SafeMethod(method string) bool {
	return method == http.MethodGet ||
		method == http.MethodHead ||
		method == http.MethodOptions
}

// RequireAuthOrigin protects cookie-authenticated mutations,
// including refresh and logout.
func (app *Uranus) RequireAuthOrigin(gc *gin.Context) bool {
	origin := gc.GetHeader("Origin")

	if origin == "" {
		if ref, err := url.Parse(gc.GetHeader("Referer")); err == nil &&
			ref.Host != "" &&
			ref.User == nil {
			origin = ref.Scheme + "://" + ref.Host
		}
	}

	if !app.AllowedAuthOrigin(origin) {
		gc.AbortWithStatusJSON(
			http.StatusForbidden,
			gin.H{"error": "forbidden origin"},
		)
		return false
	}

	return true
}
