package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// DebugMode controls verbose logging (set via AUTH_DEBUG env var)
	DebugMode = false
)

func init() {
	// Check if debug mode is enabled via environment variable
	if os.Getenv("AUTH_DEBUG") == "true" || os.Getenv("AUTH_DEBUG") == "1" {
		DebugMode = true
		log.Println("[AUTH] Debug mode enabled")
	}
}

func debugLog(format string, args ...interface{}) {
	if DebugMode {
		log.Printf(format, args...)
	}
}

// KeycloakConfig holds configuration for Keycloak JWT validation
type KeycloakConfig struct {
	RealmURL       string // e.g., "http://keycloak:8080/realms/myrealm"
	RealmPublicKey string // Optional: PEM-encoded public key (alternative to discovery)
	Issuer         string // Expected issuer (e.g., "http://keycloak:8080/realms/myrealm")
}

// KeycloakClaims extends jwt.RegisteredClaims with Keycloak-specific fields
type KeycloakClaims struct {
	jwt.RegisteredClaims
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
}

// JWKSResponse represents the JSON Web Key Set response from Keycloak
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string   `json:"kid"`
	Kty string   `json:"kty"`
	Alg string   `json:"alg"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

var (
	// TokenAuth is the JWT authentication instance
	TokenAuth *jwtauth.JWTAuth
	// keycloakPublicKey stores the RSA public key for verification
	keycloakPublicKey *rsa.PublicKey
	// jwksKeys stores all public keys from JWKS, indexed by kid
	jwksKeys map[string]*rsa.PublicKey
)

// InitKeycloakAuth initializes JWT authentication with Keycloak
// It fetches the public key from Keycloak's realm or uses a provided key
func InitKeycloakAuth(config KeycloakConfig) error {
	var err error

	debugLog("[AUTH DEBUG Initializing Keycloak auth...\n")
	debugLog("[AUTH DEBUG RealmURL: %s\n", config.RealmURL)
	debugLog("[AUTH DEBUG RealmPublicKey provided: %v\n", config.RealmPublicKey != "")

	// If a public key is provided directly, use it
	if config.RealmPublicKey != "" {
		debugLog("[AUTH DEBUG Using provided public key\n")
		keycloakPublicKey, err = parsePublicKey(config.RealmPublicKey)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
	} else if config.RealmURL != "" {
		debugLog("[AUTH DEBUG Fetching public key from JWKS endpoint\n")
		// Fetch public key from Keycloak's JWKS endpoint
		keycloakPublicKey, jwksKeys, err = fetchAllPublicKeysFromJWKS(config.RealmURL)
		if err != nil {
			return fmt.Errorf("failed to fetch public key from Keycloak: %w", err)
		}
		debugLog("[AUTH DEBUG Successfully fetched %d public keys from JWKS\n", len(jwksKeys))
	} else {
		return fmt.Errorf("either RealmURL or RealmPublicKey must be provided")
	}

	debugLog("[AUTH DEBUG Public key modulus size: %d bits\n", keycloakPublicKey.N.BitLen())

	// Create jwtauth instance with the RSA public key
	TokenAuth = jwtauth.New("RS256", nil, keycloakPublicKey)
	debugLog("[AUTH DEBUG JWT TokenAuth initialized successfully\n")

	return nil
}

// parsePublicKey parses a PEM-encoded RSA public key
func parsePublicKey(pemKey string) (*rsa.PublicKey, error) {
	// Add PEM header/footer if not present
	if !strings.HasPrefix(pemKey, "-----BEGIN") {
		pemKey = "-----BEGIN PUBLIC KEY-----\n" + pemKey + "\n-----END PUBLIC KEY-----"
	}

	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// fetchAllPublicKeysFromJWKS fetches all public keys from Keycloak's JWKS endpoint
// Returns the primary key and a map of all keys indexed by kid
func fetchAllPublicKeysFromJWKS(realmURL string) (*rsa.PublicKey, map[string]*rsa.PublicKey, error) {
	// Remove trailing slash
	realmURL = strings.TrimSuffix(realmURL, "/")
	jwksURL := realmURL + "/protocol/openid-connect/certs"

	debugLog("[AUTH DEBUG Fetching JWKS from: %s\n", jwksURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(jwksURL)
	if err != nil {
		debugLog("[AUTH DEBUG Error fetching JWKS: %v\n", err)
		return nil, nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	debugLog("[AUTH DEBUG JWKS response status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		debugLog("[AUTH DEBUG JWKS error response: %s\n", string(body))
		return nil, nil, fmt.Errorf("JWKS endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		debugLog("[AUTH DEBUG Error decoding JWKS: %v\n", err)
		return nil, nil, fmt.Errorf("failed to decode JWKS response: %w", err)
	}

	debugLog("[AUTH DEBUG Found %d keys in JWKS\n", len(jwks.Keys))

	if len(jwks.Keys) == 0 {
		return nil, nil, fmt.Errorf("no keys found in JWKS response")
	}

	// Parse all keys and store them by kid
	keyMap := make(map[string]*rsa.PublicKey)
	var primaryKey *rsa.PublicKey

	for i, k := range jwks.Keys {
		debugLog("[AUTH DEBUG Parsing key %d: kid=%s, alg=%s, use=%s\n", i, k.Kid, k.Alg, k.Use)

		var pubKey *rsa.PublicKey
		var err error

		// If x5c (certificate chain) is present, use it
		if len(k.X5c) > 0 {
			pubKey, err = parseX5C(k.X5c[0])
		} else {
			// Otherwise, construct the key from n and e
			pubKey, err = parseModulusExponent(k.N, k.E)
		}

		if err != nil {
			debugLog("[AUTH DEBUG Error parsing key %s: %v\n", k.Kid, err)
			continue
		}

		keyMap[k.Kid] = pubKey
		debugLog("[AUTH DEBUG Successfully parsed key %s\n", k.Kid)

		// Select primary key: prefer RS256 signing keys
		if primaryKey == nil || (k.Alg == "RS256" && (k.Use == "sig" || k.Use == "")) {
			primaryKey = pubKey
			debugLog("[AUTH DEBUG Set primary key to kid=%s\n", k.Kid)
		}
	}

	if primaryKey == nil {
		return nil, nil, fmt.Errorf("no valid RSA keys found in JWKS")
	}

	return primaryKey, keyMap, nil
}

// fetchPublicKeyFromJWKS fetches the public key from Keycloak's JWKS endpoint (legacy function)
func fetchPublicKeyFromJWKS(realmURL string) (*rsa.PublicKey, error) {
	primaryKey, _, err := fetchAllPublicKeysFromJWKS(realmURL)
	return primaryKey, err
}

// parseX5C parses an x5c certificate and extracts the public key
func parseX5C(x5c string) (*rsa.PublicKey, error) {
	certBytes, err := base64.StdEncoding.DecodeString(x5c)
	if err != nil {
		return nil, fmt.Errorf("failed to decode x5c: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate does not contain an RSA public key")
	}

	return rsaPub, nil
}

// parseModulusExponent constructs an RSA public key from base64url-encoded modulus and exponent
func parseModulusExponent(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert exponent bytes to int
	var e int
	for _, b := range eBytes {
		e = e*256 + int(b)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}

// Verifier is a middleware that validates JWT tokens
// Use this in your Chi router
func Verifier() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			debugLog("[AUTH DEBUG Verifier called for %s %s\n", r.Method, r.URL.Path)

			if authHeader != "" {
				// Debug decode the JWT
				DebugDecodeJWT(authHeader)
			} else {
				debugLog("[AUTH DEBUG No Authorization header present\n")
			}

			// Call the standard verifier
			jwtauth.Verifier(TokenAuth)(next).ServeHTTP(w, r)
		})
	}
}

// Authenticator is a middleware that checks if the token is valid
// It should be used after Verifier()
func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, claims, err := jwtauth.FromContext(r.Context())

		debugLog("[AUTH DEBUG Authenticator called for %s %s\n", r.Method, r.URL.Path)
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 50 {
			debugLog("[AUTH DEBUG Authorization header: %s... (truncated)\n", authHeader[:50])
		} else {
			debugLog("[AUTH DEBUG Authorization header: %s\n", authHeader)
		}

		if err != nil {
			debugLog("[AUTH DEBUG Error getting token from context: %v\n", err)
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		if token == nil {
			debugLog("[AUTH DEBUG Token is nil\n")
			http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		debugLog("[AUTH DEBUG Token type: %T\n", token)
		debugLog("[AUTH DEBUG Token claims count: %d\n", len(claims))

		// Delegate to standard authenticator
		jwtauth.Authenticator(TokenAuth)(next).ServeHTTP(w, r)
	})
}

// GetClaims extracts Keycloak claims from the request context
func GetClaims(ctx context.Context) (*KeycloakClaims, error) {
	_, claims, err := jwtauth.FromContext(ctx)
	if err != nil {
		debugLog("[AUTH DEBUG GetClaims error: %v\n", err)
		return nil, fmt.Errorf("failed to get token from context: %w", err)
	}

	debugLog("[AUTH DEBUG Raw claims from context: %+v\n", claims)

	keycloakClaims := &KeycloakClaims{}

	// Extract realm_access roles
	if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := realmAccess["roles"].([]interface{}); ok {
			keycloakClaims.RealmAccess.Roles = make([]string, 0, len(roles))
			for _, role := range roles {
				if roleStr, ok := role.(string); ok {
					keycloakClaims.RealmAccess.Roles = append(keycloakClaims.RealmAccess.Roles, roleStr)
				}
			}
		}
	}

	// Extract string fields
	if username, ok := claims["preferred_username"].(string); ok {
		keycloakClaims.PreferredUsername = username
	}
	if email, ok := claims["email"].(string); ok {
		keycloakClaims.Email = email
	}
	if name, ok := claims["name"].(string); ok {
		keycloakClaims.Name = name
	}
	if givenName, ok := claims["given_name"].(string); ok {
		keycloakClaims.GivenName = givenName
	}
	if familyName, ok := claims["family_name"].(string); ok {
		keycloakClaims.FamilyName = familyName
	}

	// Extract issuer and subject
	if iss, ok := claims["iss"].(string); ok {
		keycloakClaims.Issuer = iss
	}
	if sub, ok := claims["sub"].(string); ok {
		keycloakClaims.Subject = sub
	}

	debugLog("[AUTH DEBUG Parsed Keycloak claims - roles: %v, username: %s\n",
		keycloakClaims.RealmAccess.Roles, keycloakClaims.PreferredUsername)

	return keycloakClaims, nil
}

// GetRoles extracts the realm roles from the request context
func GetRoles(ctx context.Context) ([]string, error) {
	claims, err := GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	return claims.RealmAccess.Roles, nil
}

// GetUsername extracts the preferred username from the request context
func GetUsername(ctx context.Context) (string, error) {
	claims, err := GetClaims(ctx)
	if err != nil {
		return "", err
	}

	return claims.PreferredUsername, nil
}

// GetUserEmail extracts the user email from the request context
func GetUserEmail(ctx context.Context) (string, error) {
	claims, err := GetClaims(ctx)
	if err != nil {
		return "", err
	}

	return claims.Email, nil
}
