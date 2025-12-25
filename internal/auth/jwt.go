package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims defines JWT claims used by gridbeat.
// TokenClaims 定义 gridbeat 使用的 JWT Claims。
type TokenClaims struct {
	jwt.RegisteredClaims

	// Username is duplicated for convenience.
	// Username 为方便起见冗余保存。
	Username string `json:"username"`

	// IsRoot indicates super user.
	// IsRoot 表示是否为超级用户。
	IsRoot bool `json:"is_root"`

	// TokenType: "web" or "api".
	// TokenType：token 类型，"web" 或 "api"。
	TokenType string `json:"typ"`
}

// Sign signs a JWT with HS256.
// Sign 使用 HS256 签发 JWT。
func Sign(secret, issuer, jti string, userID uint, username string, isRoot bool, tokenType string, expiresAt *time.Time) (string, error) {
	now := time.Now()

	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   fmt.Sprintf("%s", userID),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		Username:  username,
		IsRoot:    isRoot,
		TokenType: tokenType,
	}

	if expiresAt != nil {
		claims.ExpiresAt = jwt.NewNumericDate(*expiresAt)
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return s, nil
}

// Parse verifies and parses JWT.
// Parse 验证并解析 JWT。
func Parse(secret string, tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
