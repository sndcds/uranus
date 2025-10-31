package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminGetUserProfil(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// Query the user table
	sql := strings.Replace(`
        SELECT id, email_address, display_name, first_name, last_name, locale, theme
        FROM {{schema}}.user
        WHERE id = $1`,
		"{{schema}}", h.Config.DbSchema, 1)

	var user struct {
		UserID      int     `json:"user_id"`
		Email       string  `json:"email_address"`
		DisplayName *string `json:"display_name"`
		FirstName   *string `json:"first_name"`
		LastName    *string `json:"last_name"`
		Locale      *string `json:"locale"`
		Theme       *string `json:"theme"`
	}

	row := pool.QueryRow(ctx, sql, userId)
	err := row.Scan(&user.UserID, &user.Email, &user.DisplayName, &user.FirstName, &user.LastName, &user.Locale, &user.Theme)
	if err != nil {
		if err == pgx.ErrNoRows {
			gc.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		}
		return
	}

	// Return JSON
	gc.JSON(http.StatusOK, user)
}

func (h *ApiHandler) AdminUpdateUserProfile(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// --- Parse JSON body ---
	var req struct {
		DisplayName  string `json:"display_name"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		EmailAddress string `json:"email_address"`
		Locale       string `json:"locale"`
		Theme        string `json:"theme"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid JSON: %v", err)})
		return
	}

	// --- Basic validation ---
	if !strings.Contains(req.EmailAddress, "@") {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
		return
	}

	// --- Begin transaction ---
	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// --- Check for existing email ---
	checkSQL := fmt.Sprintf(`SELECT id FROM %s.user WHERE email_address = $1`, h.Config.DbSchema)
	var existingUserID int
	err = tx.QueryRow(ctx, checkSQL, req.EmailAddress).Scan(&existingUserID)

	if err != nil && err != pgx.ErrNoRows {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to check existing email: %v", err)})
		return
	}

	if err == nil && existingUserID != userId {
		gc.JSON(http.StatusConflict, gin.H{"error": "email address already exists"})
		return
	}

	// --- Update record ---
	updateSQL := fmt.Sprintf(`
        UPDATE %s.user
        SET display_name = $1,
            first_name = $2,
            last_name = $3,
            email_address = $4,
            locale = $5,
            theme = $6
        WHERE id = $7`,
		h.Config.DbSchema)

	_, err = tx.Exec(
		ctx,
		updateSQL,
		req.DisplayName,
		req.FirstName,
		req.LastName,
		req.EmailAddress,
		req.Locale,
		req.Theme,
		userId,
	)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("update user failed: %v", err)})
		return
	}

	// --- Commit transaction ---
	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "profile updated successfully",
	})
}
