package auth

import (
	"slices"
	"net/http"
	"strings"
)

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")

		idToken, err := Verifier.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		var claims Claims
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		claims.Normalize()

		ctx := WithClaims(r.Context(), &claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			claims, ok := GetClaims(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if slices.Contains(claims.Roles, requiredRole) {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}
