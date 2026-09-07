package app

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// EncryptPassword hashes a password using bcrypt.
//
// Password policy validation, including the maximum password length,
// is intentionally handled by grains_validation.ValidatePassword.
func EncryptPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcryptCost,
	)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

// ComparePasswords compares a plaintext password with a bcrypt hash.
//
// Do not apply the application's password validation here. Login must
// verify the password exactly as supplied, including passwords from
// accounts created under an older password policy.
func ComparePasswords(storedHash, password string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(storedHash),
		[]byte(password),
	)
}
