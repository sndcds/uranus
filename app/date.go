package app

import "time"

// TODO: Review code

func IsValidDateStr(dateStr string) bool {
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}
