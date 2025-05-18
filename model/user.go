package model

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"log"
	"net/http"
)

type User struct {
	Id           int
	Name         string
	EMailAddress string
	PasswordHash string
}

func (user User) Print() {
	fmt.Println("DbUser:", user.Name, "EMailAddress:", user.EMailAddress, "PasswordHash:", user.PasswordHash)
}

func GetUserById(app app.Uranus, ctx *gin.Context, userId int) (User, error) {

	rows, err := app.MainDb.Query(context.Background(), "SELECT id, display_name, email_address FROM app.user WHERE id = $1", userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		log.Printf("Query failed: %v\n", err)
		return User{}, fmt.Errorf("user not found")
	}
	defer rows.Close()

	var user User
	for rows.Next() {
		err := rows.Scan(&user.Id, &user.Name, &user.EMailAddress)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user data"})
			return User{}, fmt.Errorf("failed to read user data")
		}
	}

	return user, nil
}

func GetUser(app *app.Uranus, ctx *gin.Context, eMail string) (User, error) {

	// Example for a sanitized/safe use of parameters.
	rows, err := app.MainDb.Query(context.Background(), "SELECT id, display_name, email_address, password_hash FROM app.user WHERE email_address = $1", eMail)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		log.Printf("Query failed: %v\n", err)
		return User{}, fmt.Errorf("user not found")
	}
	defer rows.Close()

	var user User

	for rows.Next() {
		err := rows.Scan(&user.Id, &user.Name, &user.EMailAddress, &user.PasswordHash)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user data"})
			return User{}, fmt.Errorf("failed to read user data")
		}
		log.Printf("id: %v\n", user.Id)
		log.Printf("password_hash: %v\n", user.PasswordHash)
	}

	// Check if no rows were found
	if user.PasswordHash == "" {
		return User{}, fmt.Errorf("user not found")
	}

	return user, nil
}
