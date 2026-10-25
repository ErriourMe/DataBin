package paste

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const maxPasswordBytes = 256

var ErrPasswordTooLong = errors.New("password too long")

func hashPassword(password string) (string, error) {
	if len(password) > maxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	if password == "" {
		return "", nil
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func checkPassword(hash, password string) bool {
	if hash == "" {
		return true
	}
	if password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
