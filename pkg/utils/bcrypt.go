package utils

import (
	"golang.org/x/crypto/bcrypt"
)

func EncryptPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func MatchPassword(password, encryptedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(encryptedPassword), []byte(password))
	return err == nil
}
