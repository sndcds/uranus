package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) Signup(gc *gin.Context) {
	pool := h.DbPool

	// Parse incoming JSON
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := gc.BindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	// Validate
	if !app.IsValidEmail(req.Email) {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
	}

	// Encrypt password
	passwordHash, err := app.EncryptPassword(req.Password)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt password"})
		return
	}

	// Check if user already exists
	var exists bool
	checkQuery := fmt.Sprintf(
		"SELECT EXISTS(SELECT 1 FROM %s.user WHERE email_address = $1)",
		h.Config.DbSchema)
	err = pool.QueryRow(gc, checkQuery, req.Email).Scan(&exists)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		gc.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	// Insert new user
	var newUserId int
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.user (email_address, password_hash)
		VALUES ($1, $2)
		RETURNING id`,
		h.Config.DbSchema)

	err = pool.QueryRow(gc, insertQuery, req.Email, passwordHash).Scan(&newUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Respond success
	gc.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
		"user_id": newUserId,
	})
}
