package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenGenerator struct {
	secret []byte
}

func NewTokenGenerator(secret string) *TokenGenerator {
	return &TokenGenerator{secret: []byte(secret)}
}

func (g *TokenGenerator) Generate(serverID, groupID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": serverID,
		"gID": groupID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(g.secret)
}
