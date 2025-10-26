package model

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type User struct {
	Id           int
	EMailAddress string
	PasswordHash string
	DisplayName  *string
	Locale       *string
	Theme        *string
}

func (user User) Print() {
	fmt.Println("DbUser:", user.DisplayName, "EMailAddress:", user.EMailAddress, "PasswordHash:", user.PasswordHash)
}

func GetUserById(app *app.Uranus, userId int) (User, error) {
	sqlQuery := fmt.Sprintf(
		`SELECT id, display_name, email_address, locale, theme FROM %s.user WHERE id = $1`,
		app.Config.DbSchema,
	)

	var displayName *string
	var user User

	err := app.MainDbPool.QueryRow(context.Background(), sqlQuery, userId).Scan(
		&user.Id,
		&displayName,
		&user.EMailAddress,
		&user.Locale,
		&user.Theme,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("query failed: %w", err)
	}

	// Convert *string to string
	if displayName != nil {
		user.DisplayName = displayName
	} else {
		user.DisplayName = nil
	}

	return user, nil
}

func GetUser(app *app.Uranus, eMail string) (User, error) {
	sql := fmt.Sprintf(
		"SELECT id, email_address, password_hash, display_name, locale, theme FROM %s.user WHERE email_address = $1",
		app.Config.DbSchema,
	)

	var user User
	err := app.MainDbPool.QueryRow(context.Background(), sql, eMail).Scan(
		&user.Id,
		&user.EMailAddress,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Locale,
		&user.Theme,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("query failed: %w", err)
	}

	return user, nil
}
