package api

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) Contact(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "contact")
	ctx := gc.Request.Context()
	_ = ctx // Will be used later when processing the message.

	clientIP := gc.ClientIP()
	email := strings.TrimSpace(gc.PostForm("email"))
	message := strings.TrimSpace(gc.PostForm("message"))
	website := strings.TrimSpace(gc.PostForm("website"))

	if email == "" {
		apiRequest.Required("email is required")
		return
	}

	if _, err := mail.ParseAddress(email); err != nil {
		apiRequest.Error(http.StatusBadRequest, "email is invalid")
		return
	}

	if message == "" {
		apiRequest.Required("message is required")
		return
	}

	// Honeypot.
	// For now we include it in the debug response.
	honeypotTriggered := website != ""

	// Debug response.
	apiRequest.Success(http.StatusOK, gin.H{
		"debug": gin.H{
			"ip":      clientIP,
			"email":   email,
			"message": message,
			"website": website,
			"honeypot": gin.H{
				"triggered": honeypotTriggered,
			},
			"headers": gin.H{
				"X-Real-IP":       gc.GetHeader("X-Real-IP"),
				"X-Forwarded-For": gc.GetHeader("X-Forwarded-For"),
			},
		},
	})
}
