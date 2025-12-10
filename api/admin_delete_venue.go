package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteVenue(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	venueIdStr := gc.Param("venueId")
	if venueIdStr == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Venue ID is required"})
		return
	}

	venueId, err := strconv.Atoi(venueIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid venue ID"})
		return
	}

	var body struct {
		Password string `json:"password"`
	}

	if err := gc.ShouldBindJSON(&body); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if body.Password == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
		return
	}

	var passwordHash string
	sql := fmt.Sprintf(`SELECT password_hash FROM %s.user WHERE id = $1`, h.Config.DbSchema)
	err = pool.QueryRow(ctx, sql, userId).Scan(&passwordHash)
	if err != nil {
		if err.Error() == "no rows in result set" {
			gc.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user", "details": err.Error()})
		return
	}

	if app.ComparePasswords(passwordHash, body.Password) != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.venue WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := pool.Exec(ctx, query, venueId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete venue", "details": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Venue deleted successfully", "id": venueId})
}
