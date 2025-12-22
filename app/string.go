package app

import (
	"fmt"
	"strconv"
	"strings"
)

// TODO: Review code

// TruncateAtWord truncates the string at the word boundary
func TruncateAtWord(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	words := strings.Fields(s)
	var truncated string
	for _, word := range words {
		// Add the word and a space if it doesn't exceed the max length
		if len(truncated)+len(word)+1 <= maxLength {
			if truncated == "" {
				truncated = word
			} else {
				truncated += " " + word
			}
		} else {
			break
		}
	}
	if len(truncated) < len(s) {
		truncated += " ..."
	}
	return truncated
}

func ParseIntSliceCsv(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	ids := make([]int, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %q", p)
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no valid integers provided")
	}

	return ids, nil
}

func IntSliceToCsv(ids []int) string {
	strIds := make([]string, len(ids))
	for i, id := range ids {
		strIds[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(strIds, ",")
}
