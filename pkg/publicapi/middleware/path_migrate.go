package middleware

import (
	"net/http"
)

// PathMigrate is a simple middleware which allows you to migrate old endpoints to new ones.
// It is similar to PathRewrite middleware by chi, but it has faster lookups and only does exact matches.
func PathMigrate(paths map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if newPath, ok := paths[r.URL.Path]; ok {
				r.URL.Path = newPath
			}
			next.ServeHTTP(w, r)
		})
	}
}
