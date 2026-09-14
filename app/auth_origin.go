package app

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// Keep CORS and cookie CSRF checks on the same explicit frontend allowlist.
func AllowedAuthOrigin(origin string) bool {
	return origin == "https://app.kulturbytes.de" || origin == "http://localhost:5173"
}

func SafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// RequireAuthOrigin protects cookie-authenticated mutations, including refresh
// and logout. Missing browser provenance fails closed; API clients use Bearer.
func RequireAuthOrigin(gc *gin.Context) bool {
	origin := gc.GetHeader("Origin")
	if origin == "" {
		if ref, err := url.Parse(gc.GetHeader("Referer")); err == nil && ref.Host != "" && ref.User == nil {
			origin = ref.Scheme + "://" + ref.Host
		}
	}
	if !AllowedAuthOrigin(origin) {
		gc.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden origin"})
		return false
	}
	return true
}
