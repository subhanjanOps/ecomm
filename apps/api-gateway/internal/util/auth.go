package util

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth protects endpoints using Bearer JWT (HS256)
func JWTAuth(secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				// If no secret configured, allow for local dev (but mark as 501? keep open for now)
				next(w, r)
				return
			}
			tok := readBearer(r)
			if tok == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			_, err := jwt.Parse(tok, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
}

func readBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return h[7:]
	}
	return ""
}
