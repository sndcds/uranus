package app

import (
	"golang.org/x/crypto/bcrypt"
)

// TODO: Review code

// EncryptPassword hashes a password and returns the hashed string along with any error
func EncryptPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12) // bcrypt.DefaultCost
	if err != nil {
		return "", err // Return an empty string and the error
	}
	return string(hashedPassword), nil // Return the hashed password and nil error
}

// ComparePasswords compares a plain password with a bcrypt hash
func ComparePasswords(storedHash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
}
