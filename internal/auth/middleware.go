package auth

import "net/http"

type Middleware func(http.Handler) http.Handler

func Passthrough(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
