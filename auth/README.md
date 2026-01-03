# JWT Authentication with Keycloak

This Common module provides reusable JWT authentication for Keycloak-issued tokens.

## Features

- ✅ JWT validation using Keycloak realm public keys
- ✅ Automatic JWKS endpoint discovery
- ✅ Role-based access control (RBAC) middleware
- ✅ Context helpers for extracting user info and roles
- ✅ Seamless integration with go-chi/jwtauth

## Setup

### 1. Environment Variables

Configure Keycloak connection using one of these approaches:

**Option A: Using JWKS Endpoint (Recommended)**
```bash
KEYCLOAK_REALM_URL=http://keycloak:8080/realms/myrealm
```

**Option B: Using Public Key Directly**
```bash
KEYCLOAK_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----"
```

### 2. Initialize in Your Application

```go
import "github.com/PayeTonKawa-EPSI-2025/Common-V2/auth"

func main() {
    // Initialize Keycloak auth
    err := auth.InitKeycloakAuth(auth.KeycloakConfig{
        RealmURL: os.Getenv("KEYCLOAK_REALM_URL"),
        // Or use RealmPublicKey for direct key
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### 3. Add Middleware to Chi Router

```go
import (
    "github.com/PayeTonKawa-EPSI-2025/Common-V2/auth"
    "github.com/go-chi/chi/v5"
)

func setupRouter() chi.Router {
    r := chi.NewRouter()
    
    // Public routes (no auth required)
    r.Group(func(r chi.Router) {
        r.Get("/health", healthHandler)
        r.Get("/metrics", metricsHandler)
    })
    
    // Protected routes (require valid JWT)
    r.Group(func(r chi.Router) {
        r.Use(auth.Verifier())        // Parse and verify JWT
        r.Use(auth.Authenticator)     // Ensure valid token
        
        r.Get("/customers", getCustomers)
        r.Post("/customers", createCustomer)
    })
    
    return r
}
```

## Role-Based Access Control (RBAC)

### Middleware-based RBAC

Protect routes that require specific roles:

```go
r.Group(func(r chi.Router) {
    r.Use(auth.Verifier())
    r.Use(auth.Authenticator)
    
    // Require "admin" role
    r.With(auth.RequireRole("admin")).Delete("/customers/{id}", deleteCustomer)
    
    // Require any of these roles
    r.With(auth.RequireAnyRole("admin", "manager")).Get("/reports", getReports)
    
    // Require all of these roles
    r.With(auth.RequireAllRoles("admin", "auditor")).Get("/audit-logs", getAuditLogs)
})
```

### Programmatic Role Checks

Check roles within handlers:

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    // Get all roles
    roles, err := auth.GetRoles(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Check specific role
    if auth.HasRole(r, "admin") {
        // Admin-specific logic
    }
    
    // Check any role
    if auth.HasAnyRole(r, "admin", "manager") {
        // Manager or admin logic
    }
}
```

## Extract User Information

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    // Get full Keycloak claims
    claims, err := auth.GetClaims(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    log.Printf("User: %s (%s)", claims.PreferredUsername, claims.Email)
    log.Printf("Roles: %v", claims.RealmAccess.Roles)
    
    // Or use helper functions
    username, _ := auth.GetUsername(r.Context())
    email, _ := auth.GetUserEmail(r.Context())
    roles, _ := auth.GetRoles(r.Context())
}
```

## Keycloak Token Structure

Expected JWT claims structure:

```json
{
  "exp": 1735689600,
  "iat": 1735686000,
  "jti": "abc123...",
  "iss": "http://keycloak:8080/realms/myrealm",
  "sub": "user-id-123",
  "preferred_username": "john.doe",
  "email": "john.doe@example.com",
  "name": "John Doe",
  "given_name": "John",
  "family_name": "Doe",
  "realm_access": {
    "roles": ["admin", "user", "customer-manager"]
  }
}
```

## Testing with cURL

```bash
# Get token from Keycloak
TOKEN=$(curl -X POST 'http://keycloak:8080/realms/myrealm/protocol/openid-connect/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'username=testuser' \
  -d 'password=testpass' \
  -d 'grant_type=password' \
  -d 'client_id=your-client' | jq -r '.access_token')

# Use token in API request
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/customers
```

## Error Responses

| Status | Description |
|--------|-------------|
| 401 Unauthorized | Missing or invalid JWT token |
| 403 Forbidden | Valid token but insufficient roles |

## Security Best Practices

1. **Always use HTTPS** in production
2. **Validate issuer** - The middleware checks the token issuer automatically
3. **Short token expiry** - Configure Keycloak for short-lived tokens (5-15 minutes)
4. **Refresh tokens** - Use refresh tokens for longer sessions
5. **Role principle of least privilege** - Grant minimum required roles

## Integration with Huma v2

When using with Huma, apply middleware before mounting the API:

```go
router := chi.NewRouter()

router.Group(func(r chi.Router) {
    r.Use(auth.Verifier())
    r.Use(auth.Authenticator)
    
    // Mount Huma API on protected router
    api := humachi.New(r, huma.DefaultConfig("My API", "1.0.0"))
    
    // Register routes
    huma.Register(api, huma.Operation{...}, handler)
})
```

## Troubleshooting

### "Failed to fetch JWKS"
- Check `KEYCLOAK_REALM_URL` is correct and accessible
- Verify Keycloak is running and the realm exists
- Check network connectivity between services

### "Invalid or missing token"
- Ensure `Authorization: Bearer <token>` header is present
- Verify token hasn't expired
- Check token was issued by the correct realm

### "Forbidden: requires role 'X'"
- Verify user has the required role in Keycloak
- Check role name matches exactly (case-sensitive)
- Ensure role is a realm role, not a client role (or adjust code accordingly)
