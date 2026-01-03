package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// DebugDecodeJWT decodes a JWT token without verification (for debugging only)
// This helps troubleshoot JWT structure issues
func DebugDecodeJWT(tokenString string) {
	debugLog("[AUTH DEBUG ======= JWT Debug Info =======")

	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		debugLog("[AUTH DEBUG Invalid JWT format - expected 3 parts, got %d\n", len(parts))
		return
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		debugLog("[AUTH DEBUG Error decoding header: %v\n", err)
	} else {
		var header map[string]interface{}
		if err := json.Unmarshal(headerBytes, &header); err != nil {
			debugLog("[AUTH DEBUG Error unmarshaling header: %v\n", err)
		} else {
			headerJSON, _ := json.MarshalIndent(header, "", "  ")
			debugLog("[AUTH DEBUG JWT Header:\n%s\n", string(headerJSON))

			// Highlight kid
			if kid, ok := header["kid"].(string); ok {
				debugLog("[AUTH DEBUG ★ Token kid: %s\n", kid)
				// Check if we have this key
				if jwksKeys != nil {
					if _, exists := jwksKeys[kid]; exists {
						debugLog("[AUTH DEBUG ✓ We have this key in our JWKS cache\n")
					} else {
						debugLog("[AUTH DEBUG ✗ WARNING: This kid NOT found in JWKS cache!\n")
						debugLog("[AUTH DEBUG Available kids: ")
						for k := range jwksKeys {
							log.Printf("  - %s\n", k)
						}
					}
				}
			} else {
				debugLog("[AUTH DEBUG WARNING: No kid in JWT header\n")
			}
		}
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		debugLog("[AUTH DEBUG Error decoding payload: %v\n", err)
	} else {
		var payload map[string]interface{}
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			debugLog("[AUTH DEBUG Error unmarshaling payload: %v\n", err)
		} else {
			payloadJSON, _ := json.MarshalIndent(payload, "", "  ")
			debugLog("[AUTH DEBUG JWT Payload:\n%s\n", string(payloadJSON))

			// Check for common issues
			if iss, ok := payload["iss"].(string); ok {
				debugLog("[AUTH DEBUG Issuer: %s\n", iss)
			} else {
				debugLog("[AUTH DEBUG WARNING: No issuer (iss) claim found\n")
			}

			if exp, ok := payload["exp"].(float64); ok {
				debugLog("[AUTH DEBUG Expires at: %v (unix timestamp)\n", exp)
			}

			if realmAccess, ok := payload["realm_access"].(map[string]interface{}); ok {
				if roles, ok := realmAccess["roles"].([]interface{}); ok {
					debugLog("[AUTH DEBUG Realm roles: %v\n", roles)
				}
			} else {
				debugLog("[AUTH DEBUG WARNING: No realm_access found in token\n")
			}
		}
	}

	debugLog("[AUTH DEBUG Signature (first 20 chars): %s...\n", parts[2][:min(20, len(parts[2]))])
	debugLog("[AUTH DEBUG ======= End JWT Debug =======")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EnableDebugLogging can be called to add verbose logging
// This is already enabled in the current implementation
func EnableDebugLogging() {
	debugLog("[AUTH DEBUG Debug logging is enabled")
	fmt.Println("[AUTH] Debug logging mode activated")
}
