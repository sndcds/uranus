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

// TODO: Code review

func (h *ApiHandler) AdminGetUserProfile(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	// Query the user table
	query := strings.Replace(`
        SELECT id, email_address, display_name, first_name, last_name, locale, theme
        FROM {{schema}}.user
        WHERE id = $1`,
		"{{schema}}", h.DbSchema, 1)

	var email string
	var displayName *string
	var firstName *string
	var lastName *string
	var locale *string
	var theme *string

	row := h.DbPool.QueryRow(ctx, query, userId)
	err := row.Scan(&userId, &email, &displayName, &firstName, &lastName, &locale, &theme)
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
		"user_id":       userId,
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
	userId := h.userId(gc)

	// Parse JSON body
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

	// Basic validation
	if !strings.Contains(req.EmailAddress, "@") {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
		return
	}

	// Begin transaction
	tx, err := h.DbPool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check for existing email
	checkQuery := fmt.Sprintf(`SELECT id FROM %s.user WHERE email_address = $1`, h.DbSchema)
	var existingUserId int
	err = tx.QueryRow(ctx, checkQuery, req.EmailAddress).Scan(&existingUserId)

	if err != nil && err != pgx.ErrNoRows {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to check existing email: %v", err)})
		return
	}

	if err == nil && existingUserId != userId {
		gc.JSON(http.StatusConflict, gin.H{"error": "email address already exists"})
		return
	}

	// Update record
	updateQuery := fmt.Sprintf(`
        UPDATE %s.user
        SET display_name = $1,
            first_name = $2,
            last_name = $3,
            email_address = $4,
            locale = $5,
            theme = $6
        WHERE id = $7`,
		h.DbSchema)

	_, err = tx.Exec(
		ctx,
		updateQuery,
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

	// Commit transaction
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
	userId := h.userId(gc)

	var req struct {
		Locale string `json:"locale"`
		Theme  string `json:"theme"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid JSON: %v", err)})
		return
	}

	tx, err := h.DbPool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := fmt.Sprintf(`UPDATE %s.user SET locale = $1, theme = $2 WHERE id = $3`, h.DbSchema)

	_, err = tx.Exec(
		ctx,
		query,
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
