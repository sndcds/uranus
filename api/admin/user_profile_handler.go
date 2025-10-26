package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/api"
	"github.com/sndcds/uranus/app"
)

func GetUserProfileHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool

	userId := api.UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// Query the user table
	sql := strings.Replace(`
        SELECT id, email_address, display_name, first_name, last_name, locale, theme
        FROM {{schema}}.user
        WHERE id = $1
    `, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	var user struct {
		UserID      int    `json:"user_id"`
		Email       string `json:"email_address"`
		DisplayName string `json:"display_name"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Locale      string `json:"locale"`
		Theme       string `json:"theme"`
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

func UpdateUserProfileHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool

	userId := api.UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	displayName := gc.PostForm("display_name")
	firstName := gc.PostForm("first_name")
	lastName := gc.PostForm("last_name")
	emailAddr := gc.PostForm("email_address")
	localeStr := gc.PostForm("locale")
	themeName := gc.PostForm("theme")

	// TODO: Validate email address

	// Begin DB transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	// Check if another user already has this email address
	var existingUserID int
	checkSQL := fmt.Sprintf(`SELECT id FROM %s.user WHERE email_address = $1`, app.Singleton.Config.DbSchema)
	err = tx.QueryRow(ctx, checkSQL, emailAddr).Scan(&existingUserID)
	if err != nil && err != pgx.ErrNoRows {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to check existing email: %v", err)})
		return
	}

	// If an existing record is found and it's not the same user, reject it
	if err == nil && existingUserID != userId {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusConflict, gin.H{"error": "email address already exists"})
		return
	}

	// Update existing user record
	sql := strings.Replace(`
        UPDATE {{schema}}.user
        SET display_name = $1,
            first_name = $2,
            last_name = $3,
            email_address = $4,
            locale = $5,
            theme = $6
        WHERE id = $7
    `, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	_, err = tx.Exec(
		ctx,
		sql,
		displayName,
		firstName,
		lastName,
		emailAddr,
		localeStr,
		themeName,
		userId,
	)
	if err != nil {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("update user failed: %v", err)})
		return
	}

	// Commit transaction
	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "profile updated successfully",
	})
}
