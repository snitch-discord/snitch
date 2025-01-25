package jwt

import (
	"crypto/ed25519"
	"log/slog"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenCache struct {
	token string
	mutex sync.Mutex
}

func (tokenCache *TokenCache) set(token string) {
	tokenCache.mutex.Lock()
	defer tokenCache.mutex.Unlock()
	tokenCache.token = token
}

func (tokenCache *TokenCache) Get() string {
	tokenCache.mutex.Lock()
	defer tokenCache.mutex.Unlock()
	return tokenCache.token
}

func CreateToken(key ed25519.PrivateKey) (string, error) {
	token := jwt.New(jwt.SigningMethodEdDSA)
	return token.SignedString(key)
}

func createTimedToken(key ed25519.PrivateKey, duration time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"exp": time.Now().Add(duration).Unix(),
	})

	return token.SignedString(key)
}

func startTicker(interval time.Duration, tokenCache *TokenCache, jwtGenerator func(time.Duration) (string, error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		token, err := jwtGenerator(interval)
		if err != nil {
			slog.Error("Error generating token", "Error", err)
			continue
		}
		tokenCache.set(token)
	}
}

func StartGenerator(interval time.Duration, tokenCache *TokenCache, key ed25519.PrivateKey) {
	jwtGenerator := func(time.Duration) (string, error) { return createTimedToken(key, interval) }
	firstToken, err := jwtGenerator(interval)
	if err != nil {
		slog.Error("Error generating token", "Error", err)
	}
	tokenCache.set(firstToken)

	go startTicker(interval, tokenCache, jwtGenerator)
}
