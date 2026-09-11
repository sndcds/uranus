package api

import (
	"context"
	"fmt"
)

func (h *ApiHandler) GetOrgEmail(ctx context.Context, orgUuid string) (string, error) {
	query := fmt.Sprintf(`
		SELECT contact_email
		FROM %s.organization
		WHERE uuid = $1::uuid
	`, h.DbSchema)

	var email string

	err := h.DbPool.QueryRow(ctx, query, orgUuid).Scan(&email)
	if err != nil {
		return "", err
	}

	return email, nil
}

func (h *ApiHandler) GetOrgNameAndCity(ctx context.Context, orgUuid string) (string, string, error) {
	query := fmt.Sprintf(`
		SELECT name, city
		FROM %s.organization
		WHERE uuid = $1::uuid
	`, h.DbSchema)

	var name string
	var city string

	err := h.DbPool.QueryRow(ctx, query, orgUuid).Scan(&name, city)
	if err != nil {
		return "", "", err
	}

	return name, city, nil
}
