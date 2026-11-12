package http

import (
	"crypto/subtle"
	"net/http"
)

const AuthorizationHeader = "Authorization"

func AuthMiddleware(authToken string) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get(AuthorizationHeader)
			if subtle.ConstantTimeCompare([]byte(token), []byte(authToken)) != 1 {
				WriteErrorResponse(w, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}
