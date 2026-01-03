# JWT Authentication Debugging Guide

## Debug Logs Added

The authentication system now includes comprehensive debug logging at every stage:

### 1. Initialization Logs
```
[AUTH DEBUG] Initializing Keycloak auth...
[AUTH DEBUG] RealmURL: http://localhost:8080/realms/myrealm
[AUTH DEBUG] Fetching public key from JWKS endpoint
[AUTH DEBUG] Successfully fetched public key from JWKS
[AUTH DEBUG] Public key modulus size: 2048 bits
[AUTH DEBUG] JWT TokenAuth initialized successfully
```

### 2. Request Verification Logs
```
[AUTH DEBUG] Verifier called for GET /customers
[AUTH DEBUG] ======= JWT Debug Info =======
[AUTH DEBUG] JWT Header:
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "..."
}
[AUTH DEBUG] JWT Payload:
{
  "exp": 1735689600,
  "iss": "http://localhost:8080/realms/myrealm",
  "realm_access": {
    "roles": ["admin", "user"]
  },
  ...
}
[AUTH DEBUG] ======= End JWT Debug =======
```

### 3. Authentication Logs
```
[AUTH DEBUG] Authenticator called for GET /customers
[AUTH DEBUG] Authorization header: Bearer eyJhbGc... (truncated)
[AUTH DEBUG] Token type: *jwt.Token
[AUTH DEBUG] Token claims count: 15
[AUTH DEBUG] Token authenticated successfully
```

## Common Issues and Solutions

### Issue: "Token is unauthorized"

**Possible Causes:**

#### 1. **Wrong Public Key**
Check the logs for:
```
[AUTH DEBUG] Public key modulus size: 2048 bits
```
Ensure this matches your Keycloak key size.

**Solution:**
- Verify `KEYCLOAK_REALM_URL` points to the correct realm
- Check Keycloak is accessible: `curl http://keycloak:8080/realms/myrealm/protocol/openid-connect/certs`

#### 2. **Issuer Mismatch**
Look for in the JWT payload:
```
"iss": "http://localhost:8080/realms/myrealm"
```

**Solution:**
- Keycloak URL in token must match exactly (including http/https, port)
- If using Docker, ensure the issuer in Keycloak matches how services access it
- Set Keycloak's Frontend URL if needed

#### 3. **Token Expired**
Check the exp claim:
```
"exp": 1735689600
```

**Solution:**
- Get a fresh token
- Increase token lifespan in Keycloak (Realm Settings → Tokens)

#### 4. **Algorithm Mismatch**
Header should show:
```
"alg": "RS256"
```

**Solution:**
- Keycloak must use RS256 for signing
- Check Client settings in Keycloak

#### 5. **Wrong Key ID (kid)**
If Keycloak rotated keys:
```
[AUTH DEBUG] Using key with kid: abc123, alg: RS256
```

**Solution:**
- Restart the API to fetch fresh JWKS
- Keycloak keeps old keys for grace period

## Testing Your JWT

### 1. Get a Token from Keycloak
```bash
TOKEN=$(curl -s -X POST \
  'http://localhost:8080/realms/myrealm/protocol/openid-connect/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'username=testuser' \
  -d 'password=testpass' \
  -d 'grant_type=password' \
  -d 'client_id=your-client' \
  | jq -r '.access_token')

echo "Token: $TOKEN"
```

### 2. Test the API
```bash
curl -v -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/customers
```

### 3. Decode JWT (Debugging)
Use jwt.io or:
```bash
echo $TOKEN | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

### 4. Check Server Logs
The debug logs will show exactly where authentication fails:
```bash
docker logs -f your-customer-api-container
# or
go run cmd/main.go
```

## Verifying Keycloak Configuration

### 1. Check JWKS Endpoint
```bash
curl http://localhost:8080/realms/myrealm/protocol/openid-connect/certs | jq .
```

Expected response:
```json
{
  "keys": [
    {
      "kid": "...",
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

### 2. Verify Realm is Accessible
```bash
curl http://localhost:8080/realms/myrealm/.well-known/openid-configuration
```

Should return 200 OK with configuration details.

### 3. Check Client Configuration
In Keycloak Admin Console:
- Clients → your-client → Settings
- Access Type: Should allow the grant type you're using
- Valid Redirect URIs: Configure as needed
- Service Accounts Enabled: If using client credentials

## Environment Variables

Ensure these are set:
```bash
# In .env or environment
KEYCLOAK_REALM_URL=http://localhost:8080/realms/myrealm

# Optional: Direct public key
# KEYCLOAK_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----..."
```

## Network Issues (Docker)

If running in Docker and Keycloak is also in Docker:

1. **Use Docker network name**:
   ```bash
   KEYCLOAK_REALM_URL=http://keycloak:8080/realms/myrealm
   ```

2. **Issuer must match**:
   - If token says `"iss": "http://localhost:8080/realms/myrealm"`
   - But your API uses `http://keycloak:8080`
   - → Tokens will be rejected!

3. **Solution**: Configure Keycloak Frontend URL
   ```
   Realm Settings → General → Frontend URL
   Set to: http://localhost:8080 (what clients use)
   ```

## Still Having Issues?

1. **Enable full debug logs** - Already enabled with the code updates
2. **Check the full authentication flow** in logs
3. **Compare token issuer** with what your service expects
4. **Verify algorithm** is RS256
5. **Check token hasn't expired**
6. **Ensure public key matches** the private key Keycloak used to sign

## Disabling Debug Logs (Production)

The debug logs currently use `log.Printf` which goes to stderr by default.

To disable for production, you can:
1. Remove the debug log lines
2. Use a logging level system
3. Set an environment variable to control verbosity

For now, they help diagnose the authentication issue!
