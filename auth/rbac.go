package auth

import (
	"context"
	"fmt"
	"net/http"
)

// RequireRole creates a middleware that checks if the user has the specified role
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, err := GetRoles(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !hasRole(roles, role) {
				// Generic error message - don't reveal required roles in production
				if DebugMode {
					http.Error(w, fmt.Sprintf("Forbidden: requires role '%s'", role), http.StatusForbidden)
				} else {
					http.Error(w, "Forbidden", http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole creates a middleware that checks if the user has any of the specified roles
func RequireAnyRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, err := GetRoles(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			for _, allowedRole := range allowedRoles {
				if hasRole(roles, allowedRole) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Generic error message in production
			if DebugMode {
				http.Error(w, fmt.Sprintf("Forbidden: requires one of roles: %v", allowedRoles), http.StatusForbidden)
			} else {
				http.Error(w, "Forbidden", http.StatusForbidden)
			}
		})
	}
}

// RequireAllRoles creates a middleware that checks if the user has all of the specified roles
func RequireAllRoles(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, err := GetRoles(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			for _, requiredRole := range requiredRoles {
				if !hasRole(roles, requiredRole) {
					// Generic error message in production
					if DebugMode {
						http.Error(w, fmt.Sprintf("Forbidden: requires all roles: %v", requiredRoles), http.StatusForbidden)
					} else {
						http.Error(w, "Forbidden", http.StatusForbidden)
					}
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasRole checks if a role exists in the roles slice
func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasRole checks if the user in the context has the specified role
func HasRole(r *http.Request, role string) bool {
	roles, err := GetRoles(r.Context())
	if err != nil {
		return false
	}
	return hasRole(roles, role)
}

// HasAnyRole checks if the user in the context has any of the specified roles
func HasAnyRole(r *http.Request, allowedRoles ...string) bool {
	roles, err := GetRoles(r.Context())
	if err != nil {
		return false
	}

	for _, allowedRole := range allowedRoles {
		if hasRole(roles, allowedRole) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the user in the context has all of the specified roles
func HasAllRoles(r *http.Request, requiredRoles ...string) bool {
	roles, err := GetRoles(r.Context())
	if err != nil {
		return false
	}

	for _, requiredRole := range requiredRoles {
		if !hasRole(roles, requiredRole) {
			return false
		}
	}
	return true
}

// Context-based role checking functions (for use in handlers without http.Request)

// HasRoleInContext checks if the user in the context has the specified role
func HasRoleInContext(ctx context.Context, role string) bool {
	roles, err := GetRoles(ctx)
	if err != nil {
		return false
	}
	return hasRole(roles, role)
}

// HasAnyRoleInContext checks if the user in the context has any of the specified roles
func HasAnyRoleInContext(ctx context.Context, allowedRoles ...string) bool {
	roles, err := GetRoles(ctx)
	if err != nil {
		return false
	}

	for _, allowedRole := range allowedRoles {
		if hasRole(roles, allowedRole) {
			return true
		}
	}
	return false
}

// HasAllRolesInContext checks if the user in the context has all of the specified roles
func HasAllRolesInContext(ctx context.Context, requiredRoles ...string) bool {
	roles, err := GetRoles(ctx)
	if err != nil {
		return false
	}

	for _, requiredRole := range requiredRoles {
		if !hasRole(roles, requiredRole) {
			return false
		}
	}
	return true
}
