package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: Review code

func (h *ApiHandler) AdminGetUserProfile(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := gc.GetInt("user-id")

	// Query the user table
	sql := strings.Replace(`
        SELECT id, email_address, display_name, first_name, last_name, locale, theme
        FROM {{schema}}.user
        WHERE id = $1`,
		"{{schema}}", h.Config.DbSchema, 1)

	var userID int
	var email string
	var displayName *string
	var firstName *string
	var lastName *string
	var locale *string
	var theme *string

	row := pool.QueryRow(ctx, sql, userId)
	err := row.Scan(&userID, &email, &displayName, &firstName, &lastName, &locale, &theme)
	if err != nil {
		if err == pgx.ErrNoRows {
			gc.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		}
		return
	}

	imageDir := h.Config.ProfileImageDir
	avatarFilePath := filepath.Join(imageDir, fmt.Sprintf("profile_img_%d_%d.webp", userId, 64))
	var avatarUrl *string = nil
	if _, err := os.Stat(avatarFilePath); err == nil {
		url := fmt.Sprintf("%s/api/user/%d/avatar/64", h.Config.BaseApiUrl, userId)
		avatarUrl = &url
	}

	gc.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"email_address": email,
		"display_name":  displayName,
		"first_name":    firstName,
		"last_name":     lastName,
		"locale":        locale,
		"theme":         theme,
		"avatar_url":    avatarUrl,
	})
}

func (h *ApiHandler) AdminUpdateUserProfile(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := gc.GetInt("user-id")

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

func (h *ApiHandler) AdminUpdateUserProfileSettings(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	userId := gc.GetInt("user-id")

	var req struct {
		Locale string `json:"locale"`
		Theme  string `json:"theme"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid JSON: %v", err)})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sql := fmt.Sprintf(`UPDATE %s.user SET locale = $1, theme = $2 WHERE id = $3`, h.Config.DbSchema)

	_, err = tx.Exec(
		ctx,
		sql,
		req.Locale,
		req.Theme,
		userId,
	)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("update user settings failed: %v", err)})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "user setting updated successfully",
	})
}
