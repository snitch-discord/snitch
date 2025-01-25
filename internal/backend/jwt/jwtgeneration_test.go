package jwt_test

import (
	"crypto/ed25519"
	"crypto/rand"
	snitch_jwt "snitch/internal/backend/jwt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var publicKey, privateKey, _ = ed25519.GenerateKey(rand.Reader)

func TestTokenAvailability(t *testing.T) {
	jwtDuration := time.Minute
	jwtCache := &snitch_jwt.TokenCache{}
	snitch_jwt.StartGenerator(jwtDuration, jwtCache, privateKey)

	if jwtCache.Get() == "" {
		t.Error("Token cache empty")
	}
}

func TestTokenDecode(t *testing.T) {
	jwtDuration := time.Minute
	jwtCache := &snitch_jwt.TokenCache{}
	snitch_jwt.StartGenerator(jwtDuration, jwtCache, privateKey)

	tokenString := jwtCache.Get()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}))

	if err != nil {
		t.Error(err)
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		t.Error(err)
	}

	if exp.Before(time.Now()) {
		t.Error("Token in past")
	}

	if exp.After(time.Now().Add(time.Hour)) {
		t.Error("Token too far in future")
	}
}
