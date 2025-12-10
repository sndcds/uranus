package app

import "regexp"

// TODO: Review code

func IsValidIso639_1(languageStr string) bool {
	if languageStr != "" {
		match, _ := regexp.MatchString("^[a-z]{2}$", languageStr)
		return match
	}
	return false
}
