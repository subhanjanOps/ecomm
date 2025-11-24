package util

import "net/http"

// Middleware represents an HTTP middleware that wraps a handler.
type Middleware func(http.Handler) http.Handler

// Chain composes multiple middlewares into one, applied right-to-left.
func Chain(mw ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			h = mw[i](h)
		}
		return h
	}
}

// CORSv2 is a middleware version of the permissive CORS used for dev.
func CORSv2() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// JWTAuthV2 is a middleware version of JWT auth. Currently a pass-through for dev parity.
func JWTAuthV2(secret string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// NOTE: Token validation disabled for development/testing.
			// Implement HS256 validation here when enabling auth.
			next.ServeHTTP(w, r)
		})
	}
}
