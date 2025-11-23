package middleware

import (
	"net/http"
	"strings"
)

type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	AllowCredentials bool
}

func NewCORS(opts CORSOptions) func(http.Handler) http.Handler {
	origins := make(map[string]struct{})
	for _, origin := range opts.AllowedOrigins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			origins[trimmed] = struct{}{}
		}
	}

	allowedMethods := strings.Join(orDefault(opts.AllowedMethods, []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"}), ", ")
	allowedHeaders := strings.Join(orDefault(opts.AllowedHeaders, []string{"Authorization", "Content-Type"}), ", ")
	exposeHeaders := strings.Join(opts.ExposeHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowOrigin(origin, origins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if opts.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			if exposeHeaders != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func orDefault(values, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
}

func allowOrigin(origin string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	if origin == "" {
		return false
	}
	_, ok := allowed[origin]
	return ok
}
