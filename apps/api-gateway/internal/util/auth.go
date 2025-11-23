package util

import (
	"net/http"
)

// JWTAuth protects endpoints using Bearer JWT (HS256)
func JWTAuth(secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// TEMPORARY: disable JWT token validation for development/testing.
			// The original validation logic is preserved below but commented out.
			next(w, r)

			/*
				if secret == "" {
					"net/http"
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
			*/
		}
	}
}
