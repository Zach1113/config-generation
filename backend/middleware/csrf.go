package middleware

import (
	"crypto/subtle"
	"net/http"
)

const CSRFHeaderName = "X-CSRF-Token"

func CSRFProtection(cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserContextKey).(AuthUser)
			if !ok || user.AuthMethod != AuthMethodCookie || !isUnsafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				writeError(w, http.StatusForbidden, "missing CSRF token", "csrf")
				return
			}

			header := r.Header.Get(CSRFHeaderName)
			if header == "" || subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
				writeError(w, http.StatusForbidden, "invalid CSRF token", "csrf")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}
