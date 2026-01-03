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
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/golang-jwt/jwt/v5"
)

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
)

// InitKeycloakAuth initializes JWT authentication with Keycloak
// It fetches the public key from Keycloak's realm or uses a provided key
func InitKeycloakAuth(config KeycloakConfig) error {
	var err error

	// If a public key is provided directly, use it
	if config.RealmPublicKey != "" {
		keycloakPublicKey, err = parsePublicKey(config.RealmPublicKey)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
	} else if config.RealmURL != "" {
		// Fetch public key from Keycloak's JWKS endpoint
		keycloakPublicKey, err = fetchPublicKeyFromJWKS(config.RealmURL)
		if err != nil {
			return fmt.Errorf("failed to fetch public key from Keycloak: %w", err)
		}
	} else {
		return fmt.Errorf("either RealmURL or RealmPublicKey must be provided")
	}

	// Create jwtauth instance with the RSA public key
	TokenAuth = jwtauth.New("RS256", nil, keycloakPublicKey)

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

// fetchPublicKeyFromJWKS fetches the public key from Keycloak's JWKS endpoint
func fetchPublicKeyFromJWKS(realmURL string) (*rsa.PublicKey, error) {
	// Remove trailing slash
	realmURL = strings.TrimSuffix(realmURL, "/")
	jwksURL := realmURL + "/protocol/openid-connect/certs"

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JWKS endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS response: %w", err)
	}

	if len(jwks.Keys) == 0 {
		return nil, fmt.Errorf("no keys found in JWKS response")
	}

	// Use the first key (or find by kid if needed)
	key := jwks.Keys[0]

	// If x5c (certificate chain) is present, use it
	if len(key.X5c) > 0 {
		return parseX5C(key.X5c[0])
	}

	// Otherwise, construct the key from n and e
	return parseModulusExponent(key.N, key.E)
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
	return jwtauth.Verifier(TokenAuth)
}

// Authenticator is a middleware that checks if the token is valid
// It should be used after Verifier()
func Authenticator(next http.Handler) http.Handler {
	return jwtauth.Authenticator(TokenAuth)(next)
}

// GetClaims extracts Keycloak claims from the request context
func GetClaims(ctx context.Context) (*KeycloakClaims, error) {
	_, claims, err := jwtauth.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from context: %w", err)
	}

	// Convert claims map to KeycloakClaims
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claims: %w", err)
	}

	var keycloakClaims KeycloakClaims
	if err := json.Unmarshal(claimsBytes, &keycloakClaims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return &keycloakClaims, nil
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
