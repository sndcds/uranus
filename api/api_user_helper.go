package api

import "strings"

func BuildUserLabel(
	email string,
	displayName *string,
	firstName *string,
	lastName *string,
) string {
	if displayName != nil {
		if s := strings.TrimSpace(*displayName); s != "" {
			return s
		}
	}

	var parts []string

	if firstName != nil {
		if s := strings.TrimSpace(*firstName); s != "" {
			parts = append(parts, s)
		}
	}

	if lastName != nil {
		if s := strings.TrimSpace(*lastName); s != "" {
			parts = append(parts, s)
		}
	}

	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}

	return email
}
