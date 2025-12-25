package util

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain password using bcrypt.
// HashPassword 使用 bcrypt 对明文密码进行哈希。
func HashPassword(plain string) (string, error) {
	if len(plain) < 4 {
		return "", fmt.Errorf("password too short / 密码太短")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt generate failed: %w", err)
	}
	return string(b), nil
}

// CheckPassword compares bcrypt hash and plain password.
// CheckPassword 比较 bcrypt 哈希与明文密码。
func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
