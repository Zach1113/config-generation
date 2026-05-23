package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

const (
	AuthMethodBearer = "bearer"
	AuthMethodCookie = "cookie"
)

// AuthUser represents the authenticated user extracted from the JWT.
type AuthUser struct {
	UserID     int64
	Username   string
	AuthMethod string
}

var ErrMissingToken = errors.New("missing token")

func ParseToken(tokenStr string, secret []byte, method string) (AuthUser, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return AuthUser{}, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return AuthUser{}, errors.New("invalid claims")
	}

	userIDFloat, _ := claims["user_id"].(float64)
	username, _ := claims["username"].(string)
	if userIDFloat == 0 || username == "" {
		return AuthUser{}, errors.New("missing user claims")
	}

	return AuthUser{
		UserID:     int64(userIDFloat),
		Username:   username,
		AuthMethod: method,
	}, nil
}

func AuthenticateRequest(r *http.Request, secret []byte, sessionCookieName string) (AuthUser, error) {
	if sessionCookieName != "" {
		if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			return ParseToken(cookie.Value, secret, AuthMethodCookie)
		}
	}

	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return ParseToken(strings.TrimPrefix(header, "Bearer "), secret, AuthMethodBearer)
	}

	return AuthUser{}, ErrMissingToken
}

// Auth returns middleware that validates either the session cookie or Bearer token
// and injects AuthUser into the request context.
func Auth(secret []byte, sessionCookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := AuthenticateRequest(r, secret, sessionCookieName)
			if errors.Is(err, ErrMissingToken) {
				writeError(w, http.StatusUnauthorized, "missing or malformed Authorization header", "unauthorized")
				return
			}
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token", "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// JWTAuth keeps the existing middleware constructor available for tests and
// callers that do not need to customize the cookie name.
func JWTAuth(secret []byte) func(http.Handler) http.Handler {
	return Auth(secret, "configgen_session")
}

// UserFromContext extracts the authenticated user from the request context.
// Must only be called inside authenticated routes.
func UserFromContext(ctx context.Context) AuthUser {
	return ctx.Value(UserContextKey).(AuthUser)
}
