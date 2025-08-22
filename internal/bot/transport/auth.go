package transport

import (
	"context"
	"net/http"
	"snitch/internal/bot/auth"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	serverIDContextKey = contextKey("server_id")
	groupIDContextKey  = contextKey("group_id")
)

type AuthRoundTripper struct {
	Next           http.RoundTripper
	TokenGenerator *auth.TokenGenerator
	mu             sync.Mutex
	tokenCache     map[string]string
}

func (a *AuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	serverID, ok := req.Context().Value(serverIDContextKey).(string)
	if !ok {
		return a.Next.RoundTrip(req)
	}
	groupID, ok := req.Context().Value(groupIDContextKey).(string)
	if !ok {
		return a.Next.RoundTrip(req)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if token, ok := a.tokenCache[serverID]; ok {
		// Check if token is expired
		parsedToken, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
		if err == nil {
			if exp, err := parsedToken.Claims.GetExpirationTime(); err == nil && time.Until(exp.Time) > time.Minute {
				req.Header.Set("Authorization", "Bearer "+token)
				return a.Next.RoundTrip(req)
			}
		}
	}

	token, err := a.TokenGenerator.Generate(serverID, groupID)
	if err != nil {
		return nil, err
	}
	a.tokenCache[serverID] = token
	req.Header.Set("Authorization", "Bearer "+token)
	return a.Next.RoundTrip(req)
}

func NewAuthTransport(tokenGenerator *auth.TokenGenerator, baseTransport http.RoundTripper) http.RoundTripper {
	return &AuthRoundTripper{
		Next:           baseTransport,
		TokenGenerator: tokenGenerator,
		tokenCache:     make(map[string]string),
	}
}

func WithAuthInfo(ctx context.Context, serverID, groupID string) context.Context {
	ctx = context.WithValue(ctx, serverIDContextKey, serverID)
	ctx = context.WithValue(ctx, groupIDContextKey, groupID)
	return ctx
}
