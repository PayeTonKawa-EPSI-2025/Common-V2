# Debug Mode Configuration

## Overview

The JWT authentication system now supports a production-safe debug mode that can be toggled via environment variable.

## Configuration

### Enable Debug Mode (Development)

Set the `AUTH_DEBUG` environment variable:

```bash
# In .env file or environment
AUTH_DEBUG=true
# or
AUTH_DEBUG=1
```

### Disable Debug Mode (Production)

Simply omit the variable or set it to any other value:

```bash
# Production - no debug logging
# AUTH_DEBUG=false
# or just don't set it
```

## What Changes in Production Mode?

### 1. **No Debug Logging**
When `AUTH_DEBUG` is not enabled:
- No `[AUTH DEBUG]` messages in logs
- No JWT token decoding information
- No JWKS fetching details
- No role checking traces
- Clean, minimal logs

### 2. **Generic Error Messages**
Production mode uses vague error messages to avoid information disclosure:

**Debug Mode (Development):**
```json
{
  "message": "Forbidden: requires role 'admin'"
}
```

**Production Mode:**
```json
{
  "message": "Forbidden"
}
```

This prevents attackers from discovering:
- Which roles exist in your system
- What permissions are required for routes
- Internal authorization logic

### 3. **Security Benefits**

| Information | Debug Mode | Production Mode |
|------------|-----------|-----------------|
| Required roles | ✅ Revealed | ❌ Hidden |
| JWT structure | ✅ Logged | ❌ Hidden |
| Token claims | ✅ Logged | ❌ Hidden |
| JWKS endpoint | ✅ Logged | ❌ Hidden |
| Public key details | ✅ Logged | ❌ Hidden |

## Example Logs

### Development (AUTH_DEBUG=true)

```
[AUTH] Debug mode enabled
[AUTH DEBUG] Initializing Keycloak auth...
[AUTH DEBUG] RealmURL: http://localhost:8080/realms/myrealm
[AUTH DEBUG] Fetching JWKS from: http://localhost:8080/realms/myrealm/protocol/openid-connect/certs
[AUTH DEBUG] Found 2 keys in JWKS
[AUTH DEBUG] Parsing key 0: kid=1LZ0IzPQgtka6sAZQPo6kQRJpyMuubBT1tvnUC9-3Gc, alg=RS256, use=sig
[AUTH DEBUG] Successfully parsed key 1LZ0IzPQgtka6sAZQPo6kQRJpyMuubBT1tvnUC9-3Gc
[AUTH DEBUG] Verifier called for GET /customers
[AUTH DEBUG] ======= JWT Debug Info =======
[AUTH DEBUG] JWT Header: { "alg": "RS256", "typ": "JWT", "kid": "..." }
[AUTH DEBUG] Parsed Keycloak claims - roles: [admin user], username: testuser
```

### Production (AUTH_DEBUG not set)

```
✓ Keycloak JWT authentication initialized
```

That's it! Clean and secure.

## Testing Debug Mode

### Enable for Local Development

```bash
# In your .env file
AUTH_DEBUG=true

# Run the service
go run cmd/main.go
```

### Verify Production Mode

```bash
# Remove or comment out AUTH_DEBUG
# AUTH_DEBUG=true

# Run the service
go run cmd/main.go

# You should see minimal logs
# No [AUTH DEBUG] messages
```

## Docker/Kubernetes Deployment

### Docker Compose

```yaml
services:
  customers-api:
    environment:
      - AUTH_DEBUG=false  # or omit entirely
      - KEYCLOAK_REALM_URL=https://auth.example.com/realms/prod
```

### Kubernetes

```yaml
env:
  - name: AUTH_DEBUG
    value: "false"  # or remove this env var entirely
  - name: KEYCLOAK_REALM_URL
    value: "https://auth.example.com/realms/prod"
```

## Best Practices

1. **Never enable debug mode in production** - It logs sensitive information
2. **Use generic error messages** - Already implemented when debug is off
3. **Monitor for unusual 401/403 patterns** - Sign of potential attacks
4. **Rotate keys regularly** - Debug logs may have exposed key details during development
5. **Review logs before deploying** - Ensure no debug statements leak to production

## Troubleshooting in Production

If you need to debug authentication issues in production:

1. **Don't enable AUTH_DEBUG** - Use monitoring tools instead
2. **Check metrics** - Look at 401/403 rates
3. **Verify JWKS accessibility** - Test the endpoint separately
4. **Validate token locally** - Use jwt.io to decode (never send production tokens there!)
5. **Check issuer/audience** - Common misconfiguration

## Code Changes

All debug logging now uses the `debugLog()` function:

```go
// Old (always logs)
fmt.Printf("[AUTH DEBUG] Token verified\n")

// New (only logs when AUTH_DEBUG=true)
debugLog("[AUTH DEBUG] Token verified\n")
```

This change applies to:
- `keycloak.go` - All JWT verification logic
- `debug.go` - JWT decoding utilities
- `rbac.go` - Role checking middleware
