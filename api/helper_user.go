package api

import (
	"context"
	"fmt"
)

func (h *ApiHandler) GetUserEmail(ctx context.Context, userUuid string) (string, error) {
	query := fmt.Sprintf(`
		SELECT email
		FROM %s.user
		WHERE uuid = $1::uuid
	`, h.DbSchema)

	var email string

	err := h.DbPool.QueryRow(ctx, query, userUuid).Scan(&email)
	if err != nil {
		return "", err
	}

	return email, nil
}

func (h *ApiHandler) GetUserLocale(ctx context.Context, userUuid string) (string, error) {
	query := fmt.Sprintf(`
		SELECT locale
		FROM %s.user
		WHERE uuid = $1::uuid
	`, h.DbSchema)

	var locale string

	err := h.DbPool.QueryRow(ctx, query, userUuid).Scan(&locale)
	if err != nil {
		return "", err
	}

	return locale, nil
}
