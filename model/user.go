package model

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"log"
	"net/http"
	"strings"
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

func GetUserById(app app.Uranus, gc *gin.Context, userId int) (User, error) {

	sqlTemplate := `SELECT id, display_name, email_address FROM {{schema}}.user WHERE id = $1`
	sqlQuery := strings.Replace(sqlTemplate, "{{schema}}", app.Config.DbSchema, -1)

	rows, err := app.MainDbPool.Query(context.Background(), sqlQuery, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		log.Printf("Query failed: %v\n", err)
		return User{}, fmt.Errorf("user not found")
	}
	defer rows.Close()

	var user User
	for rows.Next() {
		err := rows.Scan(&user.Id, &user.Name, &user.EMailAddress)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user data"})
			return User{}, fmt.Errorf("failed to read user data")
		}
	}

	return user, nil
}

func GetUser(app *app.Uranus, gc *gin.Context, eMail string) (User, error) {
	sqlQuery := "SELECT id, display_name, email_address, password_hash FROM " + app.Config.DbSchema + ".user WHERE email_address = $1"
	rows, err := app.MainDbPool.Query(context.Background(), sqlQuery, eMail)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		log.Printf("Query failed: %v\n", err)
		return User{}, fmt.Errorf("user not found")
	}
	defer rows.Close()

	var user User

	for rows.Next() {
		err := rows.Scan(&user.Id, &user.Name, &user.EMailAddress, &user.PasswordHash)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user data"})
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
