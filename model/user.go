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
	FirstName    *string
	LastName     *string
	DisplayName  *string
	Locale       *string
	Theme        *string
	IsActive     bool
}

func (user User) Print() {
	fmt.Println("DbUser:", user.DisplayName, "EMailAddress:", user.EMailAddress, "PasswordHash:", user.PasswordHash)
}

func GetUserById(app *app.Uranus, userId int) (User, error) {
	sql := fmt.Sprintf(
		`SELECT id, email_address, password_hash, first_name, last_name, display_name, locale, theme, is_active
		 FROM %s.user WHERE id = $1`,
		app.Config.DbSchema,
	)

	var user User
	err := app.MainDbPool.QueryRow(context.Background(), sql, userId).Scan(
		&user.Id,
		&user.EMailAddress,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.DisplayName,
		&user.Locale,
		&user.Theme,
		&user.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("query failed: %w", err)
	}

	return user, nil
}

func GetUser(app *app.Uranus, eMail string) (User, error) {
	sql := fmt.Sprintf(
		`SELECT id, email_address, password_hash, first_name, last_name, display_name, locale, theme, is_active
		WHERE email_address = $1`,
		app.Config.DbSchema,
	)

	var user User
	err := app.MainDbPool.QueryRow(context.Background(), sql, eMail).Scan(
		&user.Id,
		&user.EMailAddress,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.DisplayName,
		&user.Locale,
		&user.Theme,
		&user.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("query failed: %w", err)
	}

	return user, nil
}
