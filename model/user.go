package model

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type User struct {
	Id           int
	DisplayName  string
	EMailAddress string
	PasswordHash string
}

func (user User) Print() {
	fmt.Println("DbUser:", user.DisplayName, "EMailAddress:", user.EMailAddress, "PasswordHash:", user.PasswordHash)
}

func GetUserById(app *app.Uranus, userId int) (User, error) {
	sqlQuery := fmt.Sprintf(
		`SELECT id, display_name, email_address FROM %s.user WHERE id = $1`,
		app.Config.DbSchema,
	)

	var displayName *string
	var user User

	err := app.MainDbPool.QueryRow(context.Background(), sqlQuery, userId).Scan(
		&user.Id,
		&displayName,
		&user.EMailAddress,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("query failed: %w", err)
	}

	// Convert *string to string
	if displayName != nil {
		user.DisplayName = *displayName
	} else {
		user.DisplayName = ""
	}

	return user, nil
}

func GetUser(app *app.Uranus, eMail string) (User, error) {
	sqlQuery := fmt.Sprintf(
		"SELECT id, display_name, email_address, password_hash FROM %s.user WHERE email_address = $1",
		app.Config.DbSchema,
	)

	var displayName *string
	var user User

	err := app.MainDbPool.QueryRow(context.Background(), sqlQuery, eMail).Scan(
		&user.Id,
		&displayName,
		&user.EMailAddress,
		&user.PasswordHash,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("query failed: %w", err)
	}

	// Convert *string to string
	if displayName != nil {
		user.DisplayName = *displayName
	} else {
		user.DisplayName = ""
	}

	fmt.Println("user:", user)
	return user, nil
}
