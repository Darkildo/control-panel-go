package jwt

import (
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

type Claims struct {
	jwtlib.RegisteredClaims
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
}

func NewManager(secretKey string, tokenDuration time.Duration) *Manager {
	return &Manager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

func (m *Manager) Generate(userID int64, login string) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
		},
		UserID: userID,
		Login:  login,
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtlib.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
